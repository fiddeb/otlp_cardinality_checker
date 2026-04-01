/**
 * Fetch wrapper that validates HTTP status before parsing JSON.
 * Throws on non-2xx responses with the status code in the message.
 */
export async function fetchJSON(url, options) {
  const response = await fetch(url, options)
  if (!response.ok) {
    const text = await response.text().catch(() => '')
    let message = `HTTP ${response.status}`
    try {
      const parsed = JSON.parse(text)
      if (parsed.error) message = parsed.error
    } catch (_) {
      // not JSON — use status code message
    }
    throw new Error(message)
  }
  return response.json()
}
