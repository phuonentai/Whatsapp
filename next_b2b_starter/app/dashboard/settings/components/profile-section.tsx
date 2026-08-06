"use client";

import { UserProfile, MemberHelpers } from "@/lib/models/member.model";

interface ProfileSectionProps {
  profile: UserProfile;
}

export function ProfileSection({ profile }: ProfileSectionProps) {
  const roleConfig = MemberHelpers.getRoleConfig(profile.role);
  const displayName =
    profile.name?.trim() ||
    (profile.email ? profile.email.split("@")[0] : "AP Cash member");

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
        <header className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
            Account owner
          </p>
          <h3 className="text-2xl font-semibold text-gray-900">{displayName}</h3>
          <p className="text-sm text-gray-600">
            These details identify you across automations and approvals.
          </p>
        </header>

        <dl className="mt-8 space-y-5 text-sm">
          <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Email
            </dt>
            <dd className="text-base font-medium text-gray-900">{profile.email}</dd>
          </div>

          <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Display name
            </dt>
            <dd className="text-base font-medium text-gray-900">
              {displayName}
            </dd>
          </div>

          <div className="flex flex-col gap-2">
            <dt className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Access level
            </dt>
            <dd>
              <span
                className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium ${roleConfig.color}`}
              >
                {roleConfig.label}
              </span>
              <p className="mt-2 text-xs text-gray-500">{roleConfig.description}</p>
            </dd>
          </div>
        </dl>
      </section>

      <section className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
        <header className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">
            Workspace
          </p>
          <h3 className="text-xl font-semibold text-gray-900">
            {profile.organizationName || "No workspace connected"}
          </h3>
          <p className="text-sm text-gray-600">
            Configure branding, invite collaborators, and manage approvals within this workspace.
          </p>
        </header>

        <div className="mt-8 space-y-4 text-sm">
          <div className="rounded-xl border border-dashed border-gray-200 bg-gray-50 px-4 py-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Workspace ID
            </p>
            <p className="mt-1 font-medium text-gray-900">
              {profile.organizationId || "Not assigned"}
            </p>
            <p className="mt-2 text-xs text-gray-500">
              You&apos;ll need this ID when connecting AP Cash to external approval tools.
            </p>
          </div>
          <p className="text-xs text-gray-500">
            Need to switch workspaces or update billing ownership? Reach out to support so we can
            take care of it for you.
          </p>
        </div>
      </section>
    </div>
  );
}
