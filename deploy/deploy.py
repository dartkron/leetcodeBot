import os
import sys

import yandexcloud

from yandex.cloud.serverless.functions.v1.function_service_pb2 import ListFunctionsVersionsRequest, CreateFunctionVersionRequest
from yandex.cloud.serverless.functions.v1.function_service_pb2_grpc import FunctionServiceStub

ARCHIVE_NAME = 'bot.zip'

sa_key = {
    "id": os.getenv('service_account_key_id'),
    "service_account_id": os.getenv('SERVICE_ACCOUNT_ID'),
    "private_key": os.getenv('SERVICE_ACCOUNT_PRIVATE_KEY'),
}


sdk = yandexcloud.SDK(service_account_key=sa_key)

slService = sdk.client(FunctionServiceStub)
currentVersion = slService.ListVersions(ListFunctionsVersionsRequest(function_id=os.getenv('TARGET_FUNCTION_ID'), page_size=1)).versions[0]

if not os.path.isfile(ARCHIVE_NAME):
    print(f'Fatal error: archive {ARCHIVE_NAME} not found')
    sys.exit(1)
with open(ARCHIVE_NAME, 'rb') as f:
    content = f.read()
print('Deployment started')
createOperation = slService.CreateVersion(CreateFunctionVersionRequest(
        function_id=currentVersion.function_id,
        runtime=currentVersion.runtime,
        description='commit #asdfasd',
        entrypoint=currentVersion.entrypoint,
        resources=currentVersion.resources,
        execution_timeout=currentVersion.execution_timeout,
        service_account_id=currentVersion.service_account_id,
        content=content,
        environment={key: val for key, val in currentVersion.environment.items()},
    ))

operationResult = sdk.wait_operation_and_get_result(createOperation, timeout=300)

print('Operation finished')