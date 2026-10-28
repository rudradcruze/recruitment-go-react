const pad = (n: number) => String(n).padStart(2, "0");

// toLocalInput changes a time of the API (UTC) to the value of a datetime-local field, in the timezone of the browser.
export function toLocalInput(iso: string): string {
  const date = new Date(iso);
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

// fromLocalInput changes the value of a datetime-local field (the timezone of the browser) to a time in UTC for the API.
// An empty field gives the zero time. The API then answers 422 with the message about the Exam window.
export function fromLocalInput(value: string): string {
  return value === "" ? "0001-01-01T00:00:00Z" : new Date(value).toISOString();
}

// formatTime shows a time of the API in the timezone and the language of the browser.
export function formatTime(iso: string): string {
  return new Date(iso).toLocaleString();
}
