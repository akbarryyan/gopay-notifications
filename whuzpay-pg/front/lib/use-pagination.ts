import { useState } from "react";

function shallowChanged(prev: readonly unknown[], next: readonly unknown[]) {
  if (prev.length !== next.length) return true;
  for (let i = 0; i < prev.length; i += 1) {
    if (prev[i] !== next[i]) return true;
  }
  return false;
}

/**
 * Page-number state for list views: resets to 1 whenever any value in
 * `resetKeys` changes (e.g. filters), and clamps down to `totalPages` if the
 * current page overshoots it after the result set shrinks.
 *
 * Both adjustments happen directly during render — React's documented
 * pattern for deriving state from a changed input (see "Adjusting some
 * state when a prop changes" at https://react.dev/learn/you-might-not-need-an-effect)
 * — using a state-tracked "previous" value rather than a ref, since refs
 * may not be read/written during render. This avoids the effect+setState
 * round-trip that react-hooks/set-state-in-effect flags and that also
 * costs an extra render/fetch cycle.
 */
export function usePagination(resetKeys: readonly unknown[], totalPages: number) {
  const [page, setPage] = useState(1);
  const [prevResetKeys, setPrevResetKeys] = useState(resetKeys);

  if (shallowChanged(prevResetKeys, resetKeys)) {
    setPrevResetKeys(resetKeys);
    if (page !== 1) setPage(1);
  } else if (totalPages > 0 && page > totalPages) {
    setPage(totalPages);
  }

  return [page, setPage] as const;
}
