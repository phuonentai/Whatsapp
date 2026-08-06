// app/dashboard/settings/components/profile-tab.tsx

"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { RefreshCcw } from "lucide-react";
import { cn } from "@/lib/utils";

import { ProfileSection } from "./profile-section";
import { MemberList } from "./member-list";
import { InviteMember } from "./invite-member";
import type { OrganizationMember, UserProfile, InviteMemberRequest } from "@/lib/models/member.model";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";

interface ProfileTabProps {
  profile: UserProfile;
  members: OrganizationMember[];
  isMembersLoading: boolean;
  membersError: string | null;
  canManageMembers: boolean;
  onInvite: (request: InviteMemberRequest) => Promise<void>;
  onRefreshMembers: () => void;
}

export function ProfileTab({
  profile,
  members,
  isMembersLoading,
  membersError,
  canManageMembers,
  onInvite,
  onRefreshMembers,
}: ProfileTabProps) {
  const { hasPermission } = usePermissions();
  const canInvite = hasPermission(PERMISSIONS.ORG_MANAGE);

  const organizationId = profile.organizationId;
  const hasOrganization = Boolean(organizationId);
  const canInviteMembers = canInvite && hasOrganization;
  const canViewMembers = canManageMembers && hasOrganization;

  return (
    <div className={cn("grid gap-6", canManageMembers ? "lg:grid-cols-[360px,1fr]" : "")}>
      <Card className="self-start">
        <CardHeader className="border-b border-gray-100 pb-4">
          <CardTitle>Profile</CardTitle>
          <CardDescription>
            Personal details visible to your teammates.
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-6">
          <ProfileSection profile={profile} />
        </CardContent>
      </Card>

      {canManageMembers && (
        <div className="space-y-6">
          <Card>
            <CardHeader className="border-b border-gray-100 pb-4">
              <CardTitle>Invite a teammate</CardTitle>
              <CardDescription>
                Send a secure invitation email and assign the right role upfront.
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              {canInviteMembers ? (
                <InviteMember
                  canInvite={canInviteMembers}
                  onInvite={onInvite}
                />
              ) : (
                <Alert className="border border-amber-200 bg-amber-50">
                  <AlertTitle>Organization required</AlertTitle>
                  <AlertDescription>
                    We could not determine your organization. Please refresh and try again.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex items-start justify-between gap-3 border-b border-gray-100 pb-4">
              <div>
                <CardTitle>Team members</CardTitle>
                <CardDescription>
                  Review account access and manage team roles.
                </CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={onRefreshMembers}
                disabled={isMembersLoading || !canViewMembers}
                className="mt-1"
              >
                <RefreshCcw className="mr-2 h-4 w-4" />
                Refresh
              </Button>
            </CardHeader>
            <CardContent className="pt-6">
              {membersError && (
                <Alert
                  variant="destructive"
                  className="mb-4 border border-red-200 bg-red-50"
                >
                  <AlertTitle>Heads up</AlertTitle>
                  <AlertDescription>{membersError}</AlertDescription>
                </Alert>
              )}
              {isMembersLoading && members.length > 0 && (
                <div className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
                  <span className="inline-flex h-3 w-3 animate-spin rounded-full border-2 border-gray-200 border-t-primary-500" />
                  Refreshing team roster...
                </div>
              )}
              {canViewMembers ? (
                <MemberList
                  members={members}
                  canManage={canManageMembers}
                  currentUserId={profile.id}
                  organizationId={organizationId}
                  isFetching={isMembersLoading}
                  onMemberUpdate={onRefreshMembers}
                />
              ) : (
                <p className="text-sm text-muted-foreground">
                  Join or switch into an organization to manage members.
                </p>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
