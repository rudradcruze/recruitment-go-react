import { useEffect, useRef } from "react";

// useHeadingFocus gives a ref for the heading of a page. The person comes to the page from another page,
// so the hook moves the focus to the heading. A screen reader then reads it.
export function useHeadingFocus() {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  return heading;
}
