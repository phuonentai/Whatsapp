"use client";

import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { MoreVertical, Mail, UserMinus } from "lucide-react";
import {
  OrganizationMember,
  MemberHelpers,
  MemberRole,
} from "@/lib/models/member.model";
import { memberRepository } from "@/lib/api/api/repositories/member-repository";
import { useToast } from "@/hooks/use-toast";

interface MemberListProps {
  members: OrganizationMember[];
  canManage: boolean;
  currentUserId: string;
  organizationId: string;
  isFetching?: boolean;
  onMemberUpdate?: () => void;
}

const ROLE_OPTIONS: MemberRole[] = ["admin", "approver", "member"];

export function MemberList({
  members,
  canManage,
  currentUserId,
  organizationId,
  isFetching = false,
  onMemberUpdate,
}: MemberListProps) {
  const [pendingMemberId, setPendingMemberId] = useState<string | null>(null);
  const [roleChangingId, setRoleChangingId] = useState<string | null>(null);
  const [roleErrorMemberId, setRoleErrorMemberId] = useState<string | null>(null);
  const [roleErrorMessage, setRoleErrorMessage] = useState<string | null>(null);
  const { toast } = useToast();

  const handleRemoveMember = async (member: OrganizationMember) => {
    const memberName = member.name || member.email;
    if (!confirm(`Are you sure you want to remove ${memberName} from the organization?\n\nThis action cannot be undone.`)) {
      return;
    }

    setPendingMemberId(member.id);
    try {
      const success = await memberRepository.removeMember(member.id);
      if (success) {
        toast({
          title: "Member Removed",
          description: `${memberName} has been removed from your organization`,
        });
        onMemberUpdate?.();
      } else {
        throw new Error("Failed to remove member");
      }
    } catch (error) {
      console.error("[MemberList] Remove member error:", error);
      toast({
        title: "Error",
        description: "Failed to remove member. Please try again.",
        variant: "destructive",
      });
    } finally {
      setPendingMemberId(null);
    }
  };

  const handleResendInvite = async (memberId: string) => {
    setPendingMemberId(memberId);
    try {
      const success = await memberRepository.resendInvitation(memberId);
      if (success) {
        toast({
          title: "Success",
          description: "Invitation resent successfully",
        });
      } else {
        throw new Error("Failed to resend invitation");
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to resend invitation",
        variant: "destructive",
      });
    } finally {
      setPendingMemberId(null);
    }
  };

  const handleRoleChange = async (member: OrganizationMember, newRole: MemberRole) => {
    if (newRole === member.role) return;

    const memberName = member.name || member.email;
    const currentLabel = MemberHelpers.getRoleConfig(member.role).label;
    const targetLabel = MemberHelpers.getRoleConfig(newRole).label;
    if (!confirm(`Change ${memberName}'s role from ${currentLabel} to ${targetLabel}?`)) {
      return;
    }

    setRoleErrorMemberId(null);
    setRoleErrorMessage(null);
    setRoleChangingId(member.id);
    try {
      await memberRepository.updateMemberRole(member, newRole);
      toast({
        title: "Role updated",
        description: `${memberName} is now ${targetLabel}`,
      });
      onMemberUpdate?.();
    } catch (error) {
      console.error("[MemberList] Role change error:", error);
      const message = error instanceof Error ? error.message : "Failed to update role";
      if (/last admin|admin.*remain|demote/i.test(message)) {
        setRoleErrorMemberId(member.id);
        setRoleErrorMessage("At least one admin must remain in the organization.");
      } else {
        toast({
          title: "Error",
          description: message || "Failed to update role. Please try again.",
          variant: "destructive",
        });
      }
    } finally {
      setRoleChangingId(null);
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
      <Table>
        <TableHeader>
          <TableRow className="border-gray-200 bg-gray-50/50">
            <TableHead className="font-medium text-gray-700">Member</TableHead>
            <TableHead className="font-medium text-gray-700">Role</TableHead>
            <TableHead className="font-medium text-gray-700">Status</TableHead>
            <TableHead className="font-medium text-gray-700">Joined</TableHead>
            {canManage && <TableHead className="w-[50px]"></TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {isFetching && members.length === 0 && (
            <TableRow>
              <TableCell colSpan={canManage ? 5 : 4} className="py-8 text-center">
                <div className="flex flex-col items-center gap-2">
                  <div className="h-6 w-6 animate-spin rounded-full border-4 border-gray-200 border-t-primary-500" />
                  <p className="text-sm text-muted-foreground">Loading team members...</p>
                </div>
              </TableCell>
            </TableRow>
          )}
          {members.map((member) => {
            const roleConfig = MemberHelpers.getRoleConfig(member.role);
            const statusConfig = MemberHelpers.getStatusConfig(member.status);
            const joinedDate = MemberHelpers.formatJoinedDate(member.joinedAt);
            const isCurrentUser = member.id === currentUserId;
            const canEditRole = canManage && !isCurrentUser;
            const isRoleChanging = roleChangingId === member.id;
            const showRoleError = roleErrorMemberId === member.id && roleErrorMessage;

            return (
              <TableRow key={member.id} className="border-gray-100 hover:bg-gray-50/50 transition-colors">
                <TableCell className="py-4">
                  <div className="flex items-center gap-3">
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {member.name || member.email}
                        {isCurrentUser && (
                          <span className="ml-2 text-xs font-normal text-gray-500">(You)</span>
                        )}
                      </p>
                      {member.name && (
                        <p className="text-xs text-gray-500">{member.email}</p>
                      )}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="py-4">
                  {canEditRole ? (
                    <div className="w-40">
                      <Select
                        value={member.role}
                        onValueChange={(value) =>
                          handleRoleChange(member, value as MemberRole)
                        }
                        disabled={isRoleChanging || isFetching}
                      >
                        <SelectTrigger
                          className="h-8 w-full justify-between text-left text-sm"
                          aria-label={`Change role for ${member.name || member.email}`}
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {ROLE_OPTIONS.map((role) => {
                            const cfg = MemberHelpers.getRoleConfig(role);
                            return (
                              <SelectItem key={role} value={role}>
                                {cfg.label}
                              </SelectItem>
                            );
                          })}
                        </SelectContent>
                      </Select>
                      {showRoleError && (
                        <p className="mt-1 text-xs text-red-600">{roleErrorMessage}</p>
                      )}
                    </div>
                  ) : (
                    <>
                      <span
                        className={`inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium border ${roleConfig.color}`}
                      >
                        {roleConfig.label}
                      </span>
                      {showRoleError && (
                        <p className="mt-1 text-xs text-red-600">{roleErrorMessage}</p>
                      )}
                    </>
                  )}
                </TableCell>
                <TableCell className="py-4">
                  <span className="text-sm font-medium text-gray-700">{statusConfig.label}</span>
                </TableCell>
                <TableCell className="py-4 text-sm text-gray-600">
                  {joinedDate}
                  {member.invitedBy && (
                    <div className="text-xs text-gray-500 mt-0.5">by {member.invitedBy}</div>
                  )}
                </TableCell>
                {canManage && (
                  <TableCell className="py-4">
                    {!isCurrentUser && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={pendingMemberId === member.id || isFetching}
                            className="h-8 w-8 p-0"
                          >
                            <MoreVertical className="h-4 w-4 text-gray-500" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-48">
                          {member.status === "pending" && (
                            <DropdownMenuItem
                              disabled={pendingMemberId === member.id}
                              onClick={() => handleResendInvite(member.id)}
                              className="cursor-pointer"
                            >
                              <Mail className="mr-2 h-4 w-4" />
                              Resend Invite
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem
                            disabled={pendingMemberId === member.id}
                            onClick={() => handleRemoveMember(member)}
                            className="cursor-pointer text-red-600 focus:text-red-600"
                          >
                            <UserMinus className="mr-2 h-4 w-4" />
                            Remove Member
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </TableCell>
                )}
              </TableRow>
            );
          })}
          {!isFetching && members.length === 0 && (
            <TableRow>
              <TableCell colSpan={canManage ? 5 : 4} className="text-center py-8">
                <p className="text-sm text-muted-foreground">No members found</p>
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
