import json
import logging
from datetime import datetime, timezone
from types import SimpleNamespace

import pytest

from src.backend.aggregate_views.handler import handle_event


class FakeDynamoDB:
    def __init__(self):
        self.updates = []

    def update_item(self, **kwargs):
        self.updates.append(kwargs)


class FakeBoto3:
    def __init__(self):
        self.clients = []

    def client(self, service_name):
        client = {"service_name": service_name}
        self.clients.append(client)
        return client


def test_updates_post_view_counter_from_raw_sqs_message():
    client = FakeDynamoDB()
    event = {
        "Records": [
            {
                "messageId": "message-1",
                "body": json.dumps({"post_slug": " jenkins-aws-oidc "}),
            }
        ]
    }

    result = handle_event(
        event,
        "post-views",
        client,
        datetime(2026, 8, 14, tzinfo=timezone.utc),
    )

    assert result == {"batchItemFailures": []}
    assert client.updates[0]["TableName"] == "post-views"
    assert client.updates[0]["Key"] == {"post_slug": {"S": "jenkins-aws-oidc"}}
    assert "ADD #view_count :one" in client.updates[0]["UpdateExpression"]


def test_unwraps_sns_message_from_sqs_body():
    client = FakeDynamoDB()
    event = {
        "Records": [
            {
                "messageId": "message-1",
                "body": json.dumps(
                    {"Message": json.dumps({"post_slug": "astro-static"})}
                ),
            }
        ]
    }

    result = handle_event(event, "post-views", client)

    assert result == {"batchItemFailures": []}
    assert client.updates[0]["Key"] == {"post_slug": {"S": "astro-static"}}


def test_reports_malformed_sns_message_as_failed_record(caplog):
    client = FakeDynamoDB()
    event = {
        "Records": [
            {
                "messageId": "bad-sns",
                "body": json.dumps({"Message": "{"}),
            }
        ]
    }

    with caplog.at_level(logging.WARNING):
        result = handle_event(event, "post-views", client)

    assert result == {"batchItemFailures": [{"itemIdentifier": "bad-sns"}]}
    assert client.updates == []


def test_reports_only_failed_records(caplog):
    client = FakeDynamoDB()
    event = {
        "Records": [
            {
                "messageId": "good",
                "body": json.dumps({"post_slug": "valid-post"}),
            },
            {
                "messageId": "bad",
                "body": json.dumps({"post_slug": ""}),
            },
        ]
    }

    with caplog.at_level(logging.WARNING):
        result = handle_event(event, "post-views", client)

    assert result == {"batchItemFailures": [{"itemIdentifier": "bad"}]}
    assert len(client.updates) == 1


def test_reports_dynamodb_failure_as_failed_record(caplog):
    class FailingDynamoDB:
        def update_item(self, **_kwargs):
            raise RuntimeError("throttled")

    event = {
        "Records": [
            {
                "messageId": "retry-me",
                "body": json.dumps({"post_slug": "valid-post"}),
            }
        ]
    }

    with caplog.at_level(logging.WARNING):
        result = handle_event(event, "post-views", FailingDynamoDB())

    assert result == {"batchItemFailures": [{"itemIdentifier": "retry-me"}]}


def test_reraises_failed_record_without_message_id(caplog):
    client = FakeDynamoDB()
    event = {"Records": [{"body": json.dumps({"post_slug": ""})}]}

    with caplog.at_level(logging.WARNING), pytest.raises(ValueError):
        handle_event(event, "post-views", client)


def test_reuses_dynamodb_client_across_warm_invocations(monkeypatch):
    from src.backend.aggregate_views import handler

    fake_boto3 = FakeBoto3()
    monkeypatch.setitem(
        __import__("sys").modules,
        "boto3",
        SimpleNamespace(client=fake_boto3.client),
    )
    monkeypatch.setattr(handler, "_dynamodb_client", None)

    assert handler.get_dynamodb_client() is handler.get_dynamodb_client()
    assert [client["service_name"] for client in fake_boto3.clients] == ["dynamodb"]
