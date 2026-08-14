import json
import logging
import os
from datetime import datetime, timezone

logger = logging.getLogger(__name__)
_dynamodb_client = None


def get_dynamodb_client():
    global _dynamodb_client
    if _dynamodb_client is None:
        import boto3

        _dynamodb_client = boto3.client("dynamodb")
    return _dynamodb_client


def parse_message(record):
    body = json.loads(record["body"])

    if isinstance(body, dict) and isinstance(body.get("Message"), str):
        body = json.loads(body["Message"])

    post_slug = body.get("post_slug") if isinstance(body, dict) else None
    if not isinstance(post_slug, str) or not post_slug.strip():
        raise ValueError("missing post_slug")

    return post_slug.strip()


def update_view_count(client, table_name, post_slug, now):
    client.update_item(
        TableName=table_name,
        Key={"post_slug": {"S": post_slug}},
        UpdateExpression="SET #updated_at = :updated_at ADD #view_count :one",
        ExpressionAttributeNames={
            "#updated_at": "updated_at",
            "#view_count": "view_count",
        },
        ExpressionAttributeValues={
            ":updated_at": {"S": now.isoformat()},
            ":one": {"N": "1"},
        },
    )


def handle_event(event, table_name, client, now=None):
    failures = []
    now = now or datetime.now(timezone.utc)

    for record in event.get("Records", []):
        try:
            post_slug = parse_message(record)
            update_view_count(client, table_name, post_slug, now)
        except Exception as error:
            message_id = record.get("messageId")
            logger.warning(
                "failed to process message %s: %s",
                message_id,
                error,
            )
            if not message_id:
                raise
            failures.append({"itemIdentifier": message_id})

    return {"batchItemFailures": failures}


def lambda_handler(event, _context):
    return handle_event(
        event,
        os.environ["POST_VIEWS_TABLE_NAME"],
        get_dynamodb_client(),
    )
