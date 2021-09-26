import os
import sys
import logging

from typing import Dict, List, Any

import yandexcloud

from yandex.cloud.serverless.functions.v1.function_service_pb2 import ListFunctionsVersionsRequest, CreateFunctionVersionRequest
from yandex.cloud.serverless.functions.v1.function_service_pb2_grpc import FunctionServiceStub


FORMAT = '%(asctime)-15s %(levelname)s %(message)s'
logging.basicConfig(format=FORMAT, level=logging.INFO)
logging.info('Hello world')


sa_key = {
    "id": os.getenv('service_account_key_id'),
    "service_account_id": os.getenv('SERVICE_ACCOUNT_ID'),
    "private_key": os.getenv('SERVICE_ACCOUNT_PRIVATE_KEY'),
}


def deployFunction(targetFunctionId: str, archiveName: str, slService, sdk) -> None:
    currentVersion = slService.ListVersions(ListFunctionsVersionsRequest(function_id=targetFunctionId, page_size=1)).versions[0]

    if not os.path.isfile(archiveName):
        logging.error('Fatal error: archive %s not found', archiveName)
        sys.exit(1)
    with open(archiveName, 'rb') as f:
        content = f.read()
    logging.info('Deployment of %s started', archiveName)
    githubRef = os.getenv('GITHUB_REF')
    commitSha = os.getenv('GITHUB_SHA')
    
    # That's just a weird way to leave comment with commit in the function description
    envVars = currentVersion.environment
    envVars['GIT_VERSION'] = f'{githubRef} {commitSha}'
    
    createOperation = slService.CreateVersion(CreateFunctionVersionRequest(
            function_id=currentVersion.function_id,
            runtime=currentVersion.runtime,
            description=f'ref {githubRef} commit: {commitSha}',
            entrypoint=currentVersion.entrypoint,
            resources=currentVersion.resources,
            execution_timeout=currentVersion.execution_timeout,
            service_account_id=currentVersion.service_account_id,
            content=content,
            environment=envVars,
        ))
    sdk.wait_operation_and_get_result(createOperation, timeout=300)

    logging.info('Deployment of %s finished', archiveName)


def main() -> None:
    logging.info('Authentication at Yandex.Cloud')
    sdk = yandexcloud.SDK(service_account_key=sa_key)
    logging.info('Authentication finished')
    slService = sdk.client(FunctionServiceStub)
    deploys: List[Dict[str, Any]] = [
        {'targetFunctionId': os.getenv('TARGET_FUNCTION_ID'), 'archiveName': 'bot.zip', 'slService': slService, 'sdk': sdk},
        {'targetFunctionId': os.getenv('REMINDER_FUNCTION_ID'), 'archiveName': 'reminder.zip', 'slService': slService, 'sdk': sdk},
    ]
    for params in deploys:
        deployFunction(**params)

if __name__ == '__main__':
    main()
