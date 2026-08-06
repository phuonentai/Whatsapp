## 1. Fix auth page API base URL

- [x] 1.1 Change the default API base URL fallback in `next_b2b_starter/app/auth/page.tsx:139` from `http://localhost:8080/api` to `/api`, aligning with the pattern used by `ApiClient` in `lib/api/api/client/api-client.ts:35`
