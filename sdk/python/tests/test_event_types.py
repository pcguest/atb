from atb.event_types import (
    AI_ACTION_COMMITTED_EVENT_TYPE,
    AI_ACTION_EXECUTED_EVENT_TYPE,
    AI_ACTION_PRECOMMIT_EVENT_TYPE,
    AI_HUMAN_APPROVAL_EVENT_TYPE,
    AI_JOB_COMPLETED_EVENT_TYPE,
    AI_JOB_SCHEDULED_EVENT_TYPE,
    AI_JOB_STARTED_EVENT_TYPE,
    AI_JOB_STEP_EVENT_TYPE,
    AI_MODEL_INVOKED_EVENT_TYPE,
    AI_MODEL_OUTPUT_EVENT_TYPE,
    AI_POLICY_DECISION_EVENT_TYPE,
    AI_REQUEST_RECEIVED_EVENT_TYPE,
    AI_RESPONSE_SENT_EVENT_TYPE,
    AI_RETRIEVAL_EXECUTED_EVENT_TYPE,
    BUNDLE_ANCHOR_EVENT_TYPE,
    BUNDLE_MANIFEST_EVENT_TYPE,
    BUNDLE_SIGNATURE_EVENT_TYPE,
    DATA_EXPORT_EXECUTED_EVENT_TYPE,
    DATA_EXPORT_PRECOMMIT_EVENT_TYPE,
)


def test_profile_event_type_constants_match_go_registry() -> None:
    assert BUNDLE_MANIFEST_EVENT_TYPE == "atb.bundle.manifest"
    assert BUNDLE_ANCHOR_EVENT_TYPE == "atb.bundle.anchor"
    assert BUNDLE_SIGNATURE_EVENT_TYPE == "atb.bundle.signature"
    assert AI_REQUEST_RECEIVED_EVENT_TYPE == "ai.request.received"
    assert AI_RESPONSE_SENT_EVENT_TYPE == "ai.response.sent"
    assert AI_POLICY_DECISION_EVENT_TYPE == "ai.policy.decision"
    assert AI_RETRIEVAL_EXECUTED_EVENT_TYPE == "ai.retrieval.executed"
    assert AI_MODEL_INVOKED_EVENT_TYPE == "ai.model.invoked"
    assert AI_MODEL_OUTPUT_EVENT_TYPE == "ai.model.output"
    assert AI_ACTION_PRECOMMIT_EVENT_TYPE == "ai.action.precommit"
    assert AI_ACTION_EXECUTED_EVENT_TYPE == "ai.action.executed"
    assert AI_ACTION_COMMITTED_EVENT_TYPE == "ai.action.committed"
    assert AI_HUMAN_APPROVAL_EVENT_TYPE == "ai.human.approval"
    assert AI_JOB_SCHEDULED_EVENT_TYPE == "ai.job.scheduled"
    assert AI_JOB_STARTED_EVENT_TYPE == "ai.job.started"
    assert AI_JOB_STEP_EVENT_TYPE == "ai.job.step"
    assert AI_JOB_COMPLETED_EVENT_TYPE == "ai.job.completed"
    assert DATA_EXPORT_PRECOMMIT_EVENT_TYPE == "data.export.precommit"
    assert DATA_EXPORT_EXECUTED_EVENT_TYPE == "data.export.executed"
