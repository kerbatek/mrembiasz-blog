import json
import unittest

from src.backend.analytics_validator.handler import handle_event


class FakeSNS:
    def __init__(self):
        self.published = []

    def publish(self, **kwargs):
        self.published.append(kwargs)


class AnalyticsValidatorHandlerTest(unittest.TestCase):
    def test_publishes_valid_post_view_event(self):
        sns = FakeSNS()

        result = handle_event(
            {"pathParameters": {"slug": " astro-static "}},
            "topic-arn",
            sns,
        )

        self.assertEqual(result["statusCode"], 202)
        self.assertEqual(sns.published[0]["TopicArn"], "topic-arn")
        self.assertEqual(
            json.loads(sns.published[0]["Message"]),
            {"post_slug": "astro-static"},
        )

    def test_rejects_missing_path_parameters(self):
        sns = FakeSNS()

        result = handle_event({}, "topic-arn", sns)

        self.assertEqual(result["statusCode"], 400)
        self.assertEqual(sns.published, [])

    def test_rejects_missing_post_slug(self):
        sns = FakeSNS()

        result = handle_event({"pathParameters": {"slug": ""}}, "topic-arn", sns)

        self.assertEqual(result["statusCode"], 400)
        self.assertEqual(sns.published, [])
