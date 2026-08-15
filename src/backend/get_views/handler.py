import json
import os


def response(status_code, body):
    return {
        "statusCode": status_code,
        "headers": {
            "cache-control": "public, max-age=60",
            "content-type": "application/json",
        },
        "body": json.dumps(body),
    }


def parse_post_slug(event):
    path_parameters = event.get("pathParameters") or {}
    post_slug = path_parameters.get("slug")
    if not isinstance(post_slug, str) or not post_slug.strip():
        raise ValueError("missing slug")

    return post_slug.strip()


def handle_event(event, table_name, client):
    try:
        post_slug = parse_post_slug(event)
    except ValueError as error:
        return response(400, {"error": str(error)})

    result = client.get_item(
        TableName=table_name,
        Key={"post_slug": {"S": post_slug}},
        ProjectionExpression="view_count",
    )
    view_count = int(result.get("Item", {}).get("view_count", {}).get("N", "0"))

    return response(200, {"views": view_count})


def lambda_handler(event, _context):
    import boto3

    return handle_event(
        event,
        os.environ["POST_VIEWS_TABLE_NAME"],
        boto3.client("dynamodb"),
    )
