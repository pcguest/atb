"""Background automation profile job tracker."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from typing import TypeVar

from atb.workflow_common import WorkflowContext, new_job_id, now_rfc3339

T = TypeVar("T")


@dataclass(frozen=True)
class BackgroundJobScheduleInput:
    job_type: str
    trigger_source: str
    scheduled_by_id_hash: str
    job_id: str | None = None


class BackgroundJobTracker:
    def __init__(
        self,
        bundle=None,
        *,
        auto_save: bool = False,
        save_path: str | None = None,
        actor_id: str | None = None,
        org_id: str | None = None,
        workspace_id: str | None = None,
    ) -> None:
        self.ctx = WorkflowContext(
            bundle,
            auto_save=auto_save,
            save_path=save_path,
            actor_id=actor_id,
            org_id=org_id,
            workspace_id=workspace_id,
        )

    @property
    def bundle(self):
        return self.ctx.bundle

    def schedule(self, schedule_input: BackgroundJobScheduleInput) -> str:
        job_id = new_job_id(schedule_input.job_id)
        self.ctx.emit(
            "ai.job.scheduled",
            {
                "job_id": job_id,
                "job_type": schedule_input.job_type,
                "trigger_source": schedule_input.trigger_source,
                "scheduled_by_id_hash": schedule_input.scheduled_by_id_hash,
            },
        )
        return job_id

    def start(self, job_id: str, worker_id_hash: str, started_at: str | None = None) -> None:
        self.ctx.emit(
            "ai.job.started",
            {
                "job_id": job_id,
                "worker_id_hash": worker_id_hash,
                "started_at": started_at or now_rfc3339(),
            },
        )

    def step(self, job_id: str, step_index: int, step_type: str, step_outcome: str) -> None:
        self.ctx.emit(
            "ai.job.step",
            {
                "job_id": job_id,
                "step_index": step_index,
                "step_type": step_type,
                "step_outcome": step_outcome,
            },
        )

    def complete(self, job_id: str, outcome: str, completion_reason: str) -> None:
        self.ctx.emit(
            "ai.job.completed",
            {
                "job_id": job_id,
                "outcome": outcome,
                "completion_reason": completion_reason,
            },
        )

    def run_job(
        self,
        schedule_input: BackgroundJobScheduleInput,
        worker_id_hash: str,
        fn: Callable[[], T],
    ) -> T:
        job_id = self.schedule(schedule_input)
        self.start(job_id, worker_id_hash)
        try:
            result = fn()
            self.complete(job_id, "success", "completed")
            return result
        except Exception:
            self.complete(job_id, "error", "job failed")
            raise

    async def arun_job(
        self,
        schedule_input: BackgroundJobScheduleInput,
        worker_id_hash: str,
        fn: Callable[[], Awaitable[T]],
    ) -> T:
        job_id = self.schedule(schedule_input)
        self.start(job_id, worker_id_hash)
        try:
            result = await fn()
            self.complete(job_id, "success", "completed")
            return result
        except Exception:
            self.complete(job_id, "error", "job failed")
            raise
