function recordPostView(slug) {
  const storageKey = `post-viewed:${slug}`;

  try {
    if (localStorage.getItem(storageKey)) {
      return;
    }
  } catch {
    // Ignore storage failures; analytics should never affect reading.
  }

  fetch(`/api/views/${encodeURIComponent(slug)}`, {
    method: "POST",
    keepalive: true,
  })
    .then((response) => {
      if (response.ok) {
        try {
          localStorage.setItem(storageKey, "1");
        } catch {
          // Ignore storage failures; analytics should never affect reading.
        }
      }
    })
    .catch(() => {});
}

function loadPostViews(slug, viewElement) {
  fetch(`/api/views/${encodeURIComponent(slug)}`)
    .then((response) => (response.ok ? response.json() : null))
    .then((data) => {
      if (Number.isInteger(data?.views)) {
        viewElement.textContent = `Views: ${data.views}`;
      }
    })
    .catch(() => {});
}

const viewElement = document.getElementById("post-views");
const postSlug = viewElement?.dataset.postSlug;

if (viewElement && postSlug) {
  loadPostViews(postSlug, viewElement);
  recordPostView(postSlug);
}
