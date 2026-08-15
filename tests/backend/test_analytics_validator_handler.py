import json

from src.backend.analytics_validator.handler import handle_event


class FakeSNS:
    def __init__(self):
        self.published = []

    def publish(self, **kwargs):
        self.published.append(kwargs)


class FailingSNS:
    def publish(self, **_kwargs):
        raise RuntimeError("sns unavailable")


def test_publishes_valid_post_view_event():
    sns = FakeSNS()

    result = handle_event(
        {"pathParameters": {"slug": " astro-static "}},
        "topic-arn",
        sns,
    )

    assert result["statusCode"] == 202
    assert result["headers"] == {"content-type": "application/json"}
    assert json.loads(result["body"]) == {"accepted": True}
    assert sns.published[0]["TopicArn"] == "topic-arn"
    assert json.loads(sns.published[0]["Message"]) == {"post_slug": "astro-static"}


def test_publishes_nested_post_slug():
    sns = FakeSNS()

    result = handle_event(
        {"pathParameters": {"slug": "notes/astro-static"}},
        "topic-arn",
        sns,
    )

    assert result["statusCode"] == 202
    assert json.loads(sns.published[0]["Message"]) == {
        "post_slug": "notes/astro-static"
    }


def test_rejects_missing_path_parameters():
    sns = FakeSNS()

    result = handle_event({}, "topic-arn", sns)

    assert result["statusCode"] == 400
    assert json.loads(result["body"]) == {"error": "missing slug"}
    assert sns.published == []


def test_rejects_missing_post_slug():
    sns = FakeSNS()

    result = handle_event({"pathParameters": {"slug": ""}}, "topic-arn", sns)

    assert result["statusCode"] == 400
    assert sns.published == []


def test_publish_failure_bubbles_to_api_gateway_retry_or_500():
    try:
        handle_event(
            {"pathParameters": {"slug": "astro-static"}},
            "topic-arn",
            FailingSNS(),
        )
    except RuntimeError as error:
        assert str(error) == "sns unavailable"
    else:
        raise AssertionError("expected RuntimeError")
