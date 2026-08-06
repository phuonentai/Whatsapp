# Creating Pages

This guide shows how to create new pages and views in the app.

## Page Types

Next.js 16 uses the App Router with two component types:

- **Server Components** - Run on server, can fetch data directly
- **Client Components** - Run in browser, interactive

## Creating a Server Component Page
 
 Use generic "page.tsx" files within your route directory.
 
 **Check `app/dashboard/page.tsx` for a live example.**
 
 ```typescript
 // app/your-route/page.tsx
 import { requireMemberSession } from '@/lib/auth/stytch/server';
 
 export default async function Page() {
   const session = await requireMemberSession(); // 1. Authenticate
   // 2. Fetch data directly
   // 3. Render
   return <div>My Protected Page</div>;
 }
 ```
 
 ## Server Component Rendering Flow
 
 ```mermaid
 graph TD
     A[Request /route] --> B[middleware.ts checks auth]
     B -->|Authenticated| C[page.tsx runs on server]
     C --> D[Fetch data]
     D --> E[Render HTML]
 ```
 
 ## Creating a Client Component Page
 
 Use `'use client'` at the top of the file for interactivity.
 
 **Check `components/billing/subscription-paywall.tsx` for a client component example.**
 
 ```typescript
 // app/interactive/page.tsx
 'use client';
 
 import { usePermissions } from '@/lib/hooks/use-permissions';
 
 export default function InteractivePage() {
   const { hasPermission } = usePermissions();
   
   if (!hasPermission('resource:view')) return <div>Denied</div>;
 
   return <button onClick={() => alert('Interactive!')}>Click Me</button>;
 }
 ```
 
 ## Layouts
 
 Layouts persist across route changes and are perfect for navigation.
 
 **See `app/dashboard/layout.tsx`**
 
 ```typescript
 // app/dashboard/layout.tsx
 export default function DashboardLayout({ children }) {
   return (
     <div className="flex h-screen">
       <Sidebar />
       <main className="flex-1">{children}</main>
     </div>
   );
 }
 ```
 
 ## Protected Pages
 
 ### Middleware Protection
 Adds routes to `PROTECTED_ROUTES` in `middleware.ts`.
 
 ```typescript
 // middleware.ts
 const PROTECTED_ROUTES = ['/dashboard', '/settings'];
 ```
 
 ## Dynamic Routes
 
 Create folders with brackets like `[id]` to capture parameters.
 
 **Example**: `app/invoices/[id]/page.tsx`
 
 ```typescript
 export default function Page({ params }: { params: { id: string } }) {
   return <h1>Invoice {params.id}</h1>;
 }
 ```
 
 ## Common Patterns
 
 - **Loading**: Create `loading.tsx` for automatic skeletons.
 - **Errors**: Create `error.tsx` for error boundaries.
 - **Data Passing**: Fetch in Server Component -> Pass props to Client Component.
 
 ## Next Steps
 
 👉 **Learn about**: [Creating APIs](./07-creating-apis.md)
