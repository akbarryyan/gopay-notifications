// Single source of truth for the backend API base URL. Every module that
// needs it should import from here instead of re-declaring its own
// `process.env.NEXT_PUBLIC_API_URL ?? "..."` fallback — duplicated copies
// drift silently when the local backend port changes.
export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
