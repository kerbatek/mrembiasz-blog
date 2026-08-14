import json
import os

_sns_client = None


def get_sns_client():
    global _sns_client
    if _sns_client is None:
        import boto3

        _sns_client = boto3.client("sns")
    return _sns_client


def response(status_code, body):
    return {
        "statusCode": status_code,
        "headers": {"content-type": "application/json"},
        "body": json.dumps(body),
    }


def parse_post_slug(event):
    path_parameters = event.get("pathParameters") or {}
    post_slug = path_parameters.get("slug")
    if not isinstance(post_slug, str) or not post_slug.strip():
        raise ValueError("missing slug")

    return post_slug.strip()


def handle_event(event, topic_arn, sns_client):
    try:
        post_slug = parse_post_slug(event)
    except ValueError as error:
        return response(400, {"error": str(error)})

    sns_client.publish(
        TopicArn=topic_arn,
        Message=json.dumps({"post_slug": post_slug}),
    )

    return response(202, {"accepted": True})


def lambda_handler(event, _context):
    return handle_event(
        event,
        os.environ["ANALYTICS_TOPIC_ARN"],
        get_sns_client(),
    )
