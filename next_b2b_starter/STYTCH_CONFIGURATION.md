# Stytch Configuration Guide

This document explains how to configure Stytch B2B to prevent unknown users from receiving magic link emails and creating accounts.

## Overview

We've implemented a custom solution to address two critical security requirements:

1. **Prevent emails being sent to non-existent users**
2. **Block unknown email addresses from creating accounts**

## How It Works

### Custom Backend Validation

Instead of using Stytch's UI component directly (which always sends emails), we've created a custom flow:

1. **Frontend**: Custom email form in `app/auth/page.tsx`
2. **Backend API**: `/api/auth/magic-link` validates membership before sending
3. **Stytch API**: Only called if user is an existing member

### Security Features

✅ **Email validation**: Backend checks if email exists in any organization before sending magic link
✅ **No user enumeration**: Returns same message for existing and non-existing users
✅ **JIT provisioning blocked**: Organization settings prevent auto-creation of new members
✅ **Discovery flow restricted**: Users can only join organizations they're invited to

## Required Stytch Dashboard Configuration

### Step 1: Disable Self-Service Organization Creation

1. Log into your [Stytch Dashboard](https://stytch.com/dashboard)
2. Navigate to **Frontend SDK** settings
3. Find **"Create Organizations"** toggle under **Enabled methods**
4. **Disable** this toggle

**Result**: Users cannot create new organizations via the discovery flow

### Step 2: Configure Organization Settings (Per Organization)

For each organization in your Stytch project:

1. Navigate to **Organizations** in the dashboard
2. Select your organization
3. Go to **Settings** → **Authentication**
4. Configure the following:

```json
{
  "email_jit_provisioning": "NOT_ALLOWED",
  "email_invites": "RESTRICTED",
  "email_allowed_domains": ["your-company.com"]  // Optional: restrict by domain
}
```

**What each setting does:**

- `email_jit_provisioning: "NOT_ALLOWED"` - Prevents new members from being auto-created via magic link
- `email_invites: "RESTRICTED"` - Requires explicit invitation to join
- `email_allowed_domains` - (Optional) Only allows specific email domains

### Step 2a: (Optional) Configure Allowed Organization IDs

The backend validates emails by searching members across a specific list of organizations. To avoid an extra API call during login, you can provide a comma-separated allowlist:

```bash
STYTCH_ALLOWED_ORGANIZATION_IDS=org-test-123,org-test-456
```

If this variable is not set, we automatically fetch all organizations in the workspace and cache the IDs in memory.

### Step 3: Verify API Permissions

Ensure your Stytch API credentials have permission to:
- Search members (`organizations.members.search`)
- Send magic links (`magicLinks.email.discovery.send`)

## Testing the Implementation

### Test Case 1: Unknown Email

1. Enter an email that doesn't exist in any organization
2. Click "Send magic link"
3. **Expected**: Message says "If an account exists with that email, a magic link has been sent."
4. **Verify**: No email is actually sent
5. **Check backend logs**: Should see "No members found" for the email

### Test Case 2: Existing Member

1. Enter an email of an existing organization member
2. Click "Send magic link"
3. **Expected**: Same message as above
4. **Verify**: Email IS sent with magic link
5. **Check inbox**: Magic link email received

### Test Case 3: Magic Link Authentication

1. Click the magic link from Test Case 2
2. **Expected**: User is authenticated and redirected to dashboard
3. **Verify**: Session is created with correct organization

### Test Case 4: Unknown User Clicks Link (if they somehow got one)

1. If someone gets a magic link URL (e.g., from a legitimate user)
2. **Expected**: Authentication fails with error
3. **Verify**: No session is created, user cannot access dashboard

## API Endpoint Documentation

### POST `/api/auth/magic-link`

Validates email and sends magic link to existing members only.

**Request:**
```json
{
  "email": "user@company.com"
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "If an account exists with that email, a magic link has been sent."
}
```

**Response (Error):**
```json
{
  "error": "Unable to process request. Please try again later."
}
```

**Note**: Response is the same whether user exists or not (prevents enumeration)

## How to Add New Members

Since self-service signup is disabled, use one of these methods:

### Method 1: Invite via Stytch Dashboard

1. Go to **Organizations** → Select org → **Members**
2. Click **Invite Member**
3. Enter email and assign roles
4. User receives invitation email

### Method 2: Programmatic Invite

```typescript
import { getStytchB2BClient } from "@/lib/auth/stytch/server";

const client = getStytchB2BClient();

await client.magicLinks.email.invite.send({
  organization_id: "org-test-...",
  email_address: "newuser@company.com",
  invited_by_member_id: "member-test-...",
});
```

### Method 3: Create Member via API

```typescript
await client.organizations.members.create({
  organization_id: "org-test-...",
  email_address: "newuser@company.com",
  name: "New User",
  roles: ["member"],
});
```

## Troubleshooting

### Issue: Existing users not receiving emails

**Check:**
1. Email is verified in Stytch
2. Member status is "active" (not "pending" or "invited")
3. Backend logs for member search results
4. Stytch API credentials are correct

### Issue: Unknown users still getting emails

**Check:**
1. Using `/api/auth/magic-link` endpoint (not direct Stytch SDK call)
2. Backend search is working correctly
3. No caching issues in API route

### Issue: Users can't create organizations

**This is expected!** Self-service organization creation is disabled.

**Solution:** Create organizations manually via:
- Stytch Dashboard
- Stytch API programmatically

## Environment Variables

Required in `.env.local`:

```bash
# Stytch B2B Authentication
STYTCH_PROJECT_ID=project-test-...
STYTCH_SECRET=secret-test-...
NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN=public-token-test-...

# Session configuration
NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES=43200  # 30 days

# App URLs
NEXT_PUBLIC_APP_BASE_URL=http://localhost:3000
NEXT_PUBLIC_STYTCH_REDIRECT_PATH=/authenticate
```

## Additional Security Recommendations

1. **Enable MFA**: Require multi-factor authentication for sensitive organizations
2. **Monitor failed attempts**: Track authentication failures in your logs
3. **Rate limiting**: Add rate limiting to `/api/auth/magic-link` endpoint
4. **Email verification**: Ensure all members have verified emails
5. **Session duration**: Keep session duration appropriate for your security requirements

## Migration from Discovery Flow

If you were previously using the Discovery flow with self-service signup:

1. **Export existing members**: Get list of all current members
2. **Notify users**: Inform them that signup is now invite-only
3. **Update documentation**: Update user docs about the new auth flow
4. **Monitor support requests**: Users may try to sign up and fail

## Questions?

For Stytch-specific configuration questions:
- [Stytch B2B Documentation](https://stytch.com/docs/b2b)
- [Stytch Support](https://stytch.com/contact)

For implementation questions related to this codebase:
- Review `app/api/auth/magic-link/route.ts` for backend logic
- Review `app/auth/page.tsx` for frontend implementation
