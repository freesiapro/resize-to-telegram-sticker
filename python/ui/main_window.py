from __future__ import annotations

import os
import threading

from PySide6.QtCore import QThreadPool, Qt
from PySide6.QtWidgets import (
    QButtonGroup,
    QComboBox,
    QFileDialog,
    QGridLayout,
    QGroupBox,
    QLabel,
    QLineEdit,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QRadioButton,
    QTableWidget,
    QTableWidgetItem,
    QVBoxLayout,
    QWidget,
)

from core.selection import SelectionExpander, SelectionItem
from core.target import (
    TargetStatus,
    TargetType,
    evaluate_target,
    filter_jobs_for_target,
    summarize_jobs,
)
from infra.ffmpeg import FFmpegRunner
from infra.ffprobe import FFprobeRunner
from core.pipeline.image import ImagePipeline
from core.pipeline.video import VideoPipeline
from ui.workers import TaskWorker, WorkerContext


class MainWindow(QMainWindow):
    def __init__(self) -> None:
        super().__init__()
        self.setWindowTitle("Resize to Telegram Sticker")
        self.resize(900, 640)

        self.thread_pool = QThreadPool()
        self.cancel_event: threading.Event | None = None
        self.workers: list[TaskWorker] = []

        self.video_pipeline = VideoPipeline(probe=FFprobeRunner(), encode=FFmpegRunner())
        self.image_pipeline = ImagePipeline(encode=FFmpegRunner())

        self._skipped_count = 0
        self._total_tasks = 0
        self._completed_tasks = 0
        self._success_tasks = 0
        self._failed_tasks = 0

        self._build_ui()
        self._set_running(False)

    def _build_ui(self) -> None:
        root = QWidget()
        layout = QVBoxLayout(root)

        form = QGroupBox("Options")
        form_layout = QGridLayout(form)

        self.target_combo = QComboBox()
        self.target_values = [
            TargetType.VIDEO_STICKER,
            TargetType.STATIC_STICKER,
            TargetType.EMOJI,
        ]
        self.target_combo.addItem("Video Sticker")
        self.target_combo.addItem("Static Sticker")
        self.target_combo.addItem("Emoji")

        self.input_file_radio = QRadioButton("File")
        self.input_dir_radio = QRadioButton("Directory")
        self.input_file_radio.setChecked(True)
        mode_group = QButtonGroup(self)
        mode_group.addButton(self.input_file_radio)
        mode_group.addButton(self.input_dir_radio)

        self.input_path_edit = QLineEdit()
        self.input_browse_btn = QPushButton("Browse")
        self.input_browse_btn.clicked.connect(self._browse_input)

        self.output_dir_edit = QLineEdit("./output")
        self.output_browse_btn = QPushButton("Browse")
        self.output_browse_btn.clicked.connect(self._browse_output)

        form_layout.addWidget(QLabel("Target"), 0, 0)
        form_layout.addWidget(self.target_combo, 0, 1, 1, 2)

        form_layout.addWidget(QLabel("Input Mode"), 1, 0)
        form_layout.addWidget(self.input_file_radio, 1, 1)
        form_layout.addWidget(self.input_dir_radio, 1, 2)

        form_layout.addWidget(QLabel("Input Path"), 2, 0)
        form_layout.addWidget(self.input_path_edit, 2, 1)
        form_layout.addWidget(self.input_browse_btn, 2, 2)

        form_layout.addWidget(QLabel("Output Dir"), 3, 0)
        form_layout.addWidget(self.output_dir_edit, 3, 1)
        form_layout.addWidget(self.output_browse_btn, 3, 2)

        layout.addWidget(form)

        self.run_btn = QPushButton("Run")
        self.cancel_btn = QPushButton("Cancel")
        self.run_btn.clicked.connect(self._start_processing)
        self.cancel_btn.clicked.connect(self._cancel_processing)

        button_bar = QWidget()
        button_layout = QGridLayout(button_bar)
        button_layout.addWidget(self.run_btn, 0, 0)
        button_layout.addWidget(self.cancel_btn, 0, 1)
        button_layout.setAlignment(Qt.AlignmentFlag.AlignLeft)
        layout.addWidget(button_bar)

        self.table = QTableWidget(0, 3)
        self.table.setHorizontalHeaderLabels(["Input", "Status", "Output / Message"])
        self.table.horizontalHeader().setStretchLastSection(True)
        layout.addWidget(self.table)

        self.summary_label = QLabel("Ready.")
        layout.addWidget(self.summary_label)

        self.setCentralWidget(root)

    def _browse_input(self) -> None:
        if self.input_file_radio.isChecked():
            filters = "Media Files (*.mp4 *.mov *.webm *.mkv *.avi *.gif *.png *.jpg *.jpeg *.webp)"
            path, _ = QFileDialog.getOpenFileName(self, "Select File", "", filters)
            if path:
                self.input_path_edit.setText(path)
        else:
            path = QFileDialog.getExistingDirectory(self, "Select Directory", "")
            if path:
                self.input_path_edit.setText(path)

    def _browse_output(self) -> None:
        path = QFileDialog.getExistingDirectory(self, "Select Output Directory", "")
        if path:
            self.output_dir_edit.setText(path)

    def _start_processing(self) -> None:
        input_path = self.input_path_edit.text().strip()
        output_dir = self.output_dir_edit.text().strip() or "./output"

        if not input_path:
            QMessageBox.warning(self, "Invalid Input", "Input path is required.")
            return

        is_dir = self.input_dir_radio.isChecked()
        if is_dir and not os.path.isdir(input_path):
            QMessageBox.warning(self, "Invalid Input", "Input directory does not exist.")
            return
        if not is_dir and not os.path.isfile(input_path):
            QMessageBox.warning(self, "Invalid Input", "Input file does not exist.")
            return

        target_type = self.target_values[self.target_combo.currentIndex()]

        try:
            expander = SelectionExpander()
            expanded = expander.expand(
                [SelectionItem(path=input_path, is_dir=is_dir)], output_dir
            )
        except Exception as exc:  # noqa: BLE001
            QMessageBox.critical(self, "Scan Failed", str(exc))
            return

        hint = evaluate_target(summarize_jobs(expanded.jobs), target_type)
        if hint.status == TargetStatus.BLOCKED:
            QMessageBox.warning(self, "Invalid Selection", hint.message)
            return

        jobs = filter_jobs_for_target(expanded.jobs, target_type)
        if not jobs:
            QMessageBox.warning(self, "No Jobs", "No valid inputs for this target.")
            return

        self._skipped_count = len(expanded.skipped)
        self._total_tasks = len(jobs)
        self._completed_tasks = 0
        self._success_tasks = 0
        self._failed_tasks = 0
        self.workers = []
        self.cancel_event = threading.Event()

        self.table.setRowCount(len(jobs))
        for row, job in enumerate(jobs):
            self.table.setItem(row, 0, QTableWidgetItem(job.input_path))
            self.table.setItem(row, 1, QTableWidgetItem("Pending"))
            self.table.setItem(row, 2, QTableWidgetItem(""))

        self._update_summary()
        self._set_running(True)

        for index, job in enumerate(jobs):
            context = WorkerContext(
                job=job,
                target_type=target_type,
                video_pipeline=self.video_pipeline,
                image_pipeline=self.image_pipeline,
                cancel_event=self.cancel_event,
            )
            worker = TaskWorker(index, context)
            worker.signals.started.connect(self._on_task_started)
            worker.signals.finished.connect(self._on_task_finished)
            self.workers.append(worker)
            self.thread_pool.start(worker)

    def _cancel_processing(self) -> None:
        if self.cancel_event is not None:
            self.cancel_event.set()
        self.cancel_btn.setEnabled(False)
        self.summary_label.setText("Cancelling... Running tasks will finish.")

    def _on_task_started(self, index: int, label: str) -> None:
        self._set_status(index, "Running", "")

    def _on_task_finished(self, index: int, result) -> None:
        self._completed_tasks += 1
        if result.ok():
            self._success_tasks += 1
            self._set_status(index, "Done", result.output_path)
        else:
            self._failed_tasks += 1
            message = ""
            if result.err is not None:
                message = str(result.err)
            elif result.issues:
                message = result.issues[0].message
            self._set_status(index, "Failed", message)

        self._update_summary()
        if self._completed_tasks >= self._total_tasks:
            self._set_running(False)

    def _set_status(self, row: int, status: str, detail: str) -> None:
        self.table.setItem(row, 1, QTableWidgetItem(status))
        self.table.setItem(row, 2, QTableWidgetItem(detail))

    def _update_summary(self) -> None:
        self.summary_label.setText(
            f"Total: {self._total_tasks} | Done: {self._completed_tasks} | "
            f"Success: {self._success_tasks} | Failed: {self._failed_tasks} | "
            f"Skipped: {self._skipped_count}"
        )

    def _set_running(self, running: bool) -> None:
        self.run_btn.setEnabled(not running)
        self.cancel_btn.setEnabled(running)
        self.input_browse_btn.setEnabled(not running)
        self.output_browse_btn.setEnabled(not running)
        self.target_combo.setEnabled(not running)
        self.input_path_edit.setEnabled(not running)
        self.output_dir_edit.setEnabled(not running)
        self.input_file_radio.setEnabled(not running)
        self.input_dir_radio.setEnabled(not running)
        if not running:
            self.cancel_event = None
