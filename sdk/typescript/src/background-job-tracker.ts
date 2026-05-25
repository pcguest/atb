import {
  WorkflowContext,
  type WorkflowContextOptions,
  newJobId,
  valueDigest,
} from "./workflow-common.js";

/** Metadata for scheduling a background job. */
export interface BackgroundJobScheduleInput {
  jobType: string;
  triggerSource: string;
  scheduledByIdHash: string;
  jobId?: string;
}

export interface BackgroundJobTrackerOptions extends WorkflowContextOptions {}

/** Records background-automation profile job lifecycle events. */
export class BackgroundJobTracker {
  readonly ctx: WorkflowContext;

  constructor(options: BackgroundJobTrackerOptions = {}) {
    this.ctx = new WorkflowContext(options);
  }

  get bundle() {
    return this.ctx.bundle;
  }

  schedule(input: BackgroundJobScheduleInput): string {
    const jobId = newJobId(input.jobId);
    this.ctx.emit("ai.job.scheduled", {
      job_id: jobId,
      job_type: input.jobType,
      trigger_source: input.triggerSource,
      scheduled_by_id_hash: input.scheduledByIdHash,
    });
    return jobId;
  }

  start(jobId: string, workerIdHash: string, startedAt?: string): void {
    this.ctx.emit("ai.job.started", {
      job_id: jobId,
      worker_id_hash: workerIdHash,
      started_at: startedAt ?? nowRFC3339(),
    });
  }

  step(
    jobId: string,
    stepIndex: number,
    stepType: string,
    stepOutcome: string
  ): void {
    this.ctx.emit("ai.job.step", {
      job_id: jobId,
      step_index: stepIndex,
      step_type: stepType,
      step_outcome: stepOutcome,
    });
  }

  complete(jobId: string, outcome: string, completionReason: string): void {
    this.ctx.emit("ai.job.completed", {
      job_id: jobId,
      outcome,
      completion_reason: completionReason,
    });
  }

  async runJob<T>(
    scheduleInput: BackgroundJobScheduleInput,
    workerIdHash: string,
    fn: () => T | Promise<T>
  ): Promise<T> {
    const jobId = this.schedule(scheduleInput);
    this.start(jobId, workerIdHash);
    try {
      const result = await fn();
      this.complete(jobId, "success", "completed");
      return result;
    } catch (error) {
      this.complete(jobId, "error", "job failed");
      throw error;
    }
  }
}

function nowRFC3339(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}
