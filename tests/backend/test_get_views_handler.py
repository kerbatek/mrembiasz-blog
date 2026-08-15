import json
import unittest

from src.backend.get_views.handler import handle_event


class FakeDynamoDB:
    def __init__(self, item=None):
        self.item = item
        self.gets = []

    def get_item(self, **kwargs):
        self.gets.append(kwargs)
        return {"Item": self.item} if self.item else {}


class GetViewsHandlerTest(unittest.TestCase):
    def test_returns_post_view_count(self):
        client = FakeDynamoDB({"view_count": {"N": "42"}})

        result = handle_event(
            {"pathParameters": {"slug": " jenkins-aws-oidc "}},
            "post-views",
            client,
        )

        self.assertEqual(result["statusCode"], 200)
        self.assertEqual(json.loads(result["body"]), {"views": 42})
        self.assertEqual(client.gets[0]["TableName"], "post-views")
        self.assertEqual(
            client.gets[0]["Key"],
            {"post_slug": {"S": "jenkins-aws-oidc"}},
        )

    def test_returns_zero_for_missing_counter(self):
        result = handle_event(
            {"pathParameters": {"slug": "new-post"}},
            "post-views",
            FakeDynamoDB(),
        )

        self.assertEqual(result["statusCode"], 200)
        self.assertEqual(json.loads(result["body"]), {"views": 0})

    def test_rejects_missing_post_slug(self):
        client = FakeDynamoDB()

        result = handle_event({"pathParameters": {"slug": ""}}, "post-views", client)

        self.assertEqual(result["statusCode"], 400)
        self.assertEqual(client.gets, [])


if __name__ == "__main__":
    unittest.main()
