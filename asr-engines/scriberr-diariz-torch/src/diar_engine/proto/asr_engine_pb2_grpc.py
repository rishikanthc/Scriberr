# Generated manually to avoid requiring grpcio-tools at build time.
# Source: proto/asr_engine.proto
import grpc

from . import asr_engine_pb2 as asr_engine__pb2


class AsrEngineStub:
    def __init__(self, channel: grpc.Channel) -> None:
        self.LoadModel = channel.unary_unary(
            "/asrengine.AsrEngine/LoadModel",
            request_serializer=asr_engine__pb2.LoadModelRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.LoadModelResponse.FromString,
        )
        self.UnloadModel = channel.unary_unary(
            "/asrengine.AsrEngine/UnloadModel",
            request_serializer=asr_engine__pb2.UnloadModelRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.UnloadModelResponse.FromString,
        )
        self.StartJob = channel.unary_unary(
            "/asrengine.AsrEngine/StartJob",
            request_serializer=asr_engine__pb2.StartJobRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.StartJobResponse.FromString,
        )
        self.StopJob = channel.unary_unary(
            "/asrengine.AsrEngine/StopJob",
            request_serializer=asr_engine__pb2.StopJobRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.StopJobResponse.FromString,
        )
        self.GetJobStatus = channel.unary_unary(
            "/asrengine.AsrEngine/GetJobStatus",
            request_serializer=asr_engine__pb2.GetJobStatusRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.JobStatus.FromString,
        )
        self.StreamJobStatus = channel.unary_stream(
            "/asrengine.AsrEngine/StreamJobStatus",
            request_serializer=asr_engine__pb2.StreamJobStatusRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.JobStatus.FromString,
        )
        self.ListLoadedModels = channel.unary_unary(
            "/asrengine.AsrEngine/ListLoadedModels",
            request_serializer=asr_engine__pb2.ListLoadedModelsRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.ListLoadedModelsResponse.FromString,
        )
        self.GetEngineInfo = channel.unary_unary(
            "/asrengine.AsrEngine/GetEngineInfo",
            request_serializer=asr_engine__pb2.GetEngineInfoRequest.SerializeToString,
            response_deserializer=asr_engine__pb2.GetEngineInfoResponse.FromString,
        )


class AsrEngineServicer:
    def LoadModel(self, request, context):
        raise NotImplementedError()

    def UnloadModel(self, request, context):
        raise NotImplementedError()

    def StartJob(self, request, context):
        raise NotImplementedError()

    def StopJob(self, request, context):
        raise NotImplementedError()

    def GetJobStatus(self, request, context):
        raise NotImplementedError()

    def StreamJobStatus(self, request, context):
        raise NotImplementedError()

    def ListLoadedModels(self, request, context):
        raise NotImplementedError()

    def GetEngineInfo(self, request, context):
        raise NotImplementedError()


def add_AsrEngineServicer_to_server(servicer: AsrEngineServicer, server: grpc.Server) -> None:
    rpc_method_handlers = {
        "LoadModel": grpc.unary_unary_rpc_method_handler(
            servicer.LoadModel,
            request_deserializer=asr_engine__pb2.LoadModelRequest.FromString,
            response_serializer=asr_engine__pb2.LoadModelResponse.SerializeToString,
        ),
        "UnloadModel": grpc.unary_unary_rpc_method_handler(
            servicer.UnloadModel,
            request_deserializer=asr_engine__pb2.UnloadModelRequest.FromString,
            response_serializer=asr_engine__pb2.UnloadModelResponse.SerializeToString,
        ),
        "StartJob": grpc.unary_unary_rpc_method_handler(
            servicer.StartJob,
            request_deserializer=asr_engine__pb2.StartJobRequest.FromString,
            response_serializer=asr_engine__pb2.StartJobResponse.SerializeToString,
        ),
        "StopJob": grpc.unary_unary_rpc_method_handler(
            servicer.StopJob,
            request_deserializer=asr_engine__pb2.StopJobRequest.FromString,
            response_serializer=asr_engine__pb2.StopJobResponse.SerializeToString,
        ),
        "GetJobStatus": grpc.unary_unary_rpc_method_handler(
            servicer.GetJobStatus,
            request_deserializer=asr_engine__pb2.GetJobStatusRequest.FromString,
            response_serializer=asr_engine__pb2.JobStatus.SerializeToString,
        ),
        "StreamJobStatus": grpc.unary_stream_rpc_method_handler(
            servicer.StreamJobStatus,
            request_deserializer=asr_engine__pb2.StreamJobStatusRequest.FromString,
            response_serializer=asr_engine__pb2.JobStatus.SerializeToString,
        ),
        "ListLoadedModels": grpc.unary_unary_rpc_method_handler(
            servicer.ListLoadedModels,
            request_deserializer=asr_engine__pb2.ListLoadedModelsRequest.FromString,
            response_serializer=asr_engine__pb2.ListLoadedModelsResponse.SerializeToString,
        ),
        "GetEngineInfo": grpc.unary_unary_rpc_method_handler(
            servicer.GetEngineInfo,
            request_deserializer=asr_engine__pb2.GetEngineInfoRequest.FromString,
            response_serializer=asr_engine__pb2.GetEngineInfoResponse.SerializeToString,
        ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
        "asrengine.AsrEngine", rpc_method_handlers
    )
    server.add_generic_rpc_handlers((generic_handler,))


__all__ = [
    "AsrEngineStub",
    "AsrEngineServicer",
    "add_AsrEngineServicer_to_server",
]
