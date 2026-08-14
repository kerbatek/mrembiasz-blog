import json
import unittest
from datetime import datetime, timezone

from src.backend.aggregate_views.handler import handle_event


class FakeDynamoDB:
    def __init__(self):
        self.updates = []

    def update_item(self, **kwargs):
        self.updates.append(kwargs)


class AggregateViewsHandlerTest(unittest.TestCase):
    def test_updates_post_view_counter_from_raw_sqs_message(self):
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

        self.assertEqual(result, {"batchItemFailures": []})
        self.assertEqual(client.updates[0]["TableName"], "post-views")
        self.assertEqual(
            client.updates[0]["Key"],
            {"post_slug": {"S": "jenkins-aws-oidc"}},
        )
        self.assertIn("ADD #view_count :one", client.updates[0]["UpdateExpression"])

    def test_unwraps_sns_message_from_sqs_body(self):
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

        self.assertEqual(result, {"batchItemFailures": []})
        self.assertEqual(
            client.updates[0]["Key"],
            {"post_slug": {"S": "astro-static"}},
        )

    def test_reports_only_failed_records(self):
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

        with self.assertLogs("src.backend.aggregate_views.handler", level="WARNING"):
            result = handle_event(event, "post-views", client)

        self.assertEqual(result, {"batchItemFailures": [{"itemIdentifier": "bad"}]})
        self.assertEqual(len(client.updates), 1)

    def test_reraises_failed_record_without_message_id(self):
        client = FakeDynamoDB()
        event = {"Records": [{"body": json.dumps({"post_slug": ""})}]}

        with self.assertRaises(ValueError):
            handle_event(event, "post-views", client)


if __name__ == "__main__":
    unittest.main()
