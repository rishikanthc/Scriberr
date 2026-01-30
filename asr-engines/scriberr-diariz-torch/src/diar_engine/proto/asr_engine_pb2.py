# Generated manually to avoid requiring grpcio-tools at build time.
# Source: proto/asr_engine.proto
from google.protobuf import descriptor_pb2 as _descriptor_pb2
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database

_sym_db = _symbol_database.Default()


def _build_file_descriptor_proto() -> _descriptor_pb2.FileDescriptorProto:
    fd = _descriptor_pb2.FileDescriptorProto()
    fd.name = "asr_engine.proto"
    fd.package = "asrengine"
    fd.syntax = "proto3"

    # Enum JobState
    enum = fd.enum_type.add()
    enum.name = "JobState"
    for name, number in [
        ("JOB_STATE_UNSPECIFIED", 0),
        ("JOB_STATE_QUEUED", 1),
        ("JOB_STATE_RUNNING", 2),
        ("JOB_STATE_COMPLETED", 3),
        ("JOB_STATE_FAILED", 4),
        ("JOB_STATE_CANCELLED", 5),
    ]:
        value = enum.value.add()
        value.name = name
        value.number = number

    # Message: ModelSpec
    msg = fd.message_type.add()
    msg.name = "ModelSpec"
    fields = [
        ("model_id", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("model_name", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("model_path", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("providers", 4, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_REPEATED, None),
        ("intra_op_threads", 5, _descriptor_pb2.FieldDescriptorProto.TYPE_INT32, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("vad_backend", 6, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
    ]
    for name, number, ftype, label, type_name in fields:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = label
        if type_name:
            f.type_name = type_name

    # Message: LoadModelRequest
    msg = fd.message_type.add()
    msg.name = "LoadModelRequest"
    f = msg.field.add()
    f.name = "spec"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_MESSAGE
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL
    f.type_name = ".asrengine.ModelSpec"

    # Message: LoadModelResponse
    msg = fd.message_type.add()
    msg.name = "LoadModelResponse"
    for name, number, ftype in [
        ("model_id", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
        ("ok", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_BOOL),
        ("message", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: UnloadModelRequest
    msg = fd.message_type.add()
    msg.name = "UnloadModelRequest"
    f = msg.field.add()
    f.name = "model_id"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: UnloadModelResponse
    msg = fd.message_type.add()
    msg.name = "UnloadModelResponse"
    for name, number, ftype in [
        ("ok", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_BOOL),
        ("message", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: StartJobRequest with map params
    msg = fd.message_type.add()
    msg.name = "StartJobRequest"
    params_entry = msg.nested_type.add()
    params_entry.name = "ParamsEntry"
    params_entry.options.map_entry = True
    f = params_entry.field.add()
    f.name = "key"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL
    f = params_entry.field.add()
    f.name = "value"
    f.number = 2
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    for name, number, ftype, label, type_name in [
        ("job_id", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("input_path", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("output_dir", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("model_id", 4, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("params", 5, _descriptor_pb2.FieldDescriptorProto.TYPE_MESSAGE, _descriptor_pb2.FieldDescriptorProto.LABEL_REPEATED, ".asrengine.StartJobRequest.ParamsEntry"),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = label
        if type_name:
            f.type_name = type_name

    # Message: StartJobResponse
    msg = fd.message_type.add()
    msg.name = "StartJobResponse"
    for name, number, ftype in [
        ("job_id", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
        ("accepted", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_BOOL),
        ("message", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: StopJobRequest
    msg = fd.message_type.add()
    msg.name = "StopJobRequest"
    f = msg.field.add()
    f.name = "job_id"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: StopJobResponse
    msg = fd.message_type.add()
    msg.name = "StopJobResponse"
    for name, number, ftype in [
        ("ok", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_BOOL),
        ("message", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: GetJobStatusRequest
    msg = fd.message_type.add()
    msg.name = "GetJobStatusRequest"
    f = msg.field.add()
    f.name = "job_id"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: StreamJobStatusRequest
    msg = fd.message_type.add()
    msg.name = "StreamJobStatusRequest"
    f = msg.field.add()
    f.name = "job_id"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Message: JobStatus with map outputs
    msg = fd.message_type.add()
    msg.name = "JobStatus"
    outputs_entry = msg.nested_type.add()
    outputs_entry.name = "OutputsEntry"
    outputs_entry.options.map_entry = True
    f = outputs_entry.field.add()
    f.name = "key"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL
    f = outputs_entry.field.add()
    f.name = "value"
    f.number = 2
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_STRING
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    for name, number, ftype, label, type_name in [
        ("job_id", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("state", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_ENUM, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, ".asrengine.JobState"),
        ("message", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("progress", 4, _descriptor_pb2.FieldDescriptorProto.TYPE_DOUBLE, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("outputs", 5, _descriptor_pb2.FieldDescriptorProto.TYPE_MESSAGE, _descriptor_pb2.FieldDescriptorProto.LABEL_REPEATED, ".asrengine.JobStatus.OutputsEntry"),
        ("started_unix_ms", 6, _descriptor_pb2.FieldDescriptorProto.TYPE_INT64, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
        ("finished_unix_ms", 7, _descriptor_pb2.FieldDescriptorProto.TYPE_INT64, _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL, None),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = label
        if type_name:
            f.type_name = type_name

    # Message: ListLoadedModelsRequest
    msg = fd.message_type.add()
    msg.name = "ListLoadedModelsRequest"

    # Message: ListLoadedModelsResponse
    msg = fd.message_type.add()
    msg.name = "ListLoadedModelsResponse"
    f = msg.field.add()
    f.name = "models"
    f.number = 1
    f.type = _descriptor_pb2.FieldDescriptorProto.TYPE_MESSAGE
    f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_REPEATED
    f.type_name = ".asrengine.ModelSpec"

    # Message: GetEngineInfoRequest
    msg = fd.message_type.add()
    msg.name = "GetEngineInfoRequest"

    # Message: GetEngineInfoResponse
    msg = fd.message_type.add()
    msg.name = "GetEngineInfoResponse"
    for name, number, ftype in [
        ("busy", 1, _descriptor_pb2.FieldDescriptorProto.TYPE_BOOL),
        ("active_job_id", 2, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
        ("loaded_model_id", 3, _descriptor_pb2.FieldDescriptorProto.TYPE_STRING),
        ("rss_bytes", 4, _descriptor_pb2.FieldDescriptorProto.TYPE_INT64),
    ]:
        f = msg.field.add()
        f.name = name
        f.number = number
        f.type = ftype
        f.label = _descriptor_pb2.FieldDescriptorProto.LABEL_OPTIONAL

    # Service: AsrEngine
    svc = fd.service.add()
    svc.name = "AsrEngine"
    for name, input_type, output_type, server_streaming in [
        ("LoadModel", ".asrengine.LoadModelRequest", ".asrengine.LoadModelResponse", False),
        ("UnloadModel", ".asrengine.UnloadModelRequest", ".asrengine.UnloadModelResponse", False),
        ("StartJob", ".asrengine.StartJobRequest", ".asrengine.StartJobResponse", False),
        ("StopJob", ".asrengine.StopJobRequest", ".asrengine.StopJobResponse", False),
        ("GetJobStatus", ".asrengine.GetJobStatusRequest", ".asrengine.JobStatus", False),
        ("StreamJobStatus", ".asrengine.StreamJobStatusRequest", ".asrengine.JobStatus", True),
        ("ListLoadedModels", ".asrengine.ListLoadedModelsRequest", ".asrengine.ListLoadedModelsResponse", False),
        ("GetEngineInfo", ".asrengine.GetEngineInfoRequest", ".asrengine.GetEngineInfoResponse", False),
    ]:
        m = svc.method.add()
        m.name = name
        m.input_type = input_type
        m.output_type = output_type
        m.server_streaming = server_streaming

    return fd


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(
    _build_file_descriptor_proto().SerializeToString()
)

from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper

JobState = _enum_type_wrapper.EnumTypeWrapper(
    DESCRIPTOR.enum_types_by_name["JobState"]
)
JOB_STATE_UNSPECIFIED = 0
JOB_STATE_QUEUED = 1
JOB_STATE_RUNNING = 2
JOB_STATE_COMPLETED = 3
JOB_STATE_FAILED = 4
JOB_STATE_CANCELLED = 5

ModelSpec = _reflection.GeneratedProtocolMessageType(
    "ModelSpec",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["ModelSpec"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(ModelSpec)

LoadModelRequest = _reflection.GeneratedProtocolMessageType(
    "LoadModelRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["LoadModelRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(LoadModelRequest)

LoadModelResponse = _reflection.GeneratedProtocolMessageType(
    "LoadModelResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["LoadModelResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(LoadModelResponse)

UnloadModelRequest = _reflection.GeneratedProtocolMessageType(
    "UnloadModelRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["UnloadModelRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(UnloadModelRequest)

UnloadModelResponse = _reflection.GeneratedProtocolMessageType(
    "UnloadModelResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["UnloadModelResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(UnloadModelResponse)

StartJobRequest = _reflection.GeneratedProtocolMessageType(
    "StartJobRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["StartJobRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(StartJobRequest)

StartJobResponse = _reflection.GeneratedProtocolMessageType(
    "StartJobResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["StartJobResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(StartJobResponse)

StopJobRequest = _reflection.GeneratedProtocolMessageType(
    "StopJobRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["StopJobRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(StopJobRequest)

StopJobResponse = _reflection.GeneratedProtocolMessageType(
    "StopJobResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["StopJobResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(StopJobResponse)

GetJobStatusRequest = _reflection.GeneratedProtocolMessageType(
    "GetJobStatusRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["GetJobStatusRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(GetJobStatusRequest)

StreamJobStatusRequest = _reflection.GeneratedProtocolMessageType(
    "StreamJobStatusRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["StreamJobStatusRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(StreamJobStatusRequest)

JobStatus = _reflection.GeneratedProtocolMessageType(
    "JobStatus",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["JobStatus"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(JobStatus)

ListLoadedModelsRequest = _reflection.GeneratedProtocolMessageType(
    "ListLoadedModelsRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["ListLoadedModelsRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(ListLoadedModelsRequest)

ListLoadedModelsResponse = _reflection.GeneratedProtocolMessageType(
    "ListLoadedModelsResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["ListLoadedModelsResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(ListLoadedModelsResponse)

GetEngineInfoRequest = _reflection.GeneratedProtocolMessageType(
    "GetEngineInfoRequest",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["GetEngineInfoRequest"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(GetEngineInfoRequest)

GetEngineInfoResponse = _reflection.GeneratedProtocolMessageType(
    "GetEngineInfoResponse",
    (_message.Message,),
    {"DESCRIPTOR": DESCRIPTOR.message_types_by_name["GetEngineInfoResponse"], "__module__": "asr_engine.proto.asr_engine_pb2"},
)
_sym_db.RegisterMessage(GetEngineInfoResponse)

__all__ = [
    "ModelSpec",
    "LoadModelRequest",
    "LoadModelResponse",
    "UnloadModelRequest",
    "UnloadModelResponse",
    "StartJobRequest",
    "StartJobResponse",
    "StopJobRequest",
    "StopJobResponse",
    "GetJobStatusRequest",
    "StreamJobStatusRequest",
    "JobStatus",
    "ListLoadedModelsRequest",
    "ListLoadedModelsResponse",
    "GetEngineInfoRequest",
    "GetEngineInfoResponse",
    "JobState",
]
