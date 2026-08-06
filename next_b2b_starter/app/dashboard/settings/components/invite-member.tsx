"use client";

import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Send, CheckCircle2 } from "lucide-react";
import { MemberRole } from "@/lib/models/member.model";
import { useToast } from "@/hooks/use-toast";
import { rbacRepository } from "@/lib/api/api/repositories/rbac-repository";

interface InviteMemberProps {
  canInvite: boolean;
  onInvite: (request: {
    email: string;
    name: string;
    role: MemberRole;
    sendEmail?: boolean;
  }) => Promise<void>;
}

interface RoleOption {
  id: MemberRole;
  name: string;
  description: string;
  typicalUsers: string;
}

// Default roles matching the backend RBAC system
// Used as fallback if API call fails
const DEFAULT_ROLES: RoleOption[] = [
  {
    id: "member",
    name: "Member",
    description: "Basic access - can view and create resources",
    typicalUsers: "Team members, staff",
  },
  {
    id: "manager",
    name: "Manager",
    description: "Elevated access - can edit, delete, approve resources and view organization",
    typicalUsers: "Team leads, supervisors, managers",
  },
  {
    id: "admin",
    name: "Admin",
    description: "Full system control - all permissions and organization management",
    typicalUsers: "Directors, system administrators",
  },
];

export function InviteMember({
  canInvite,
  onInvite,
}: InviteMemberProps) {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState<MemberRole>("member");
  const [isLoading, setIsLoading] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);
  const [invitedEmail, setInvitedEmail] = useState("");
  const [roleOptions, setRoleOptions] = useState<RoleOption[]>([]);
  const [isLoadingRoles, setIsLoadingRoles] = useState(false);
  const [rolesError, setRolesError] = useState<string | null>(null);
  const { toast } = useToast();
  const [selectPortalContainer, setSelectPortalContainer] = useState<HTMLElement | null>(null);

  useEffect(() => {
    const container = document.getElementById("invite-member-dialog");
    setSelectPortalContainer(container);
  }, []);

  useEffect(() => {
    let mounted = true;
    const loadRoles = async () => {
      setIsLoadingRoles(true);
      setRolesError(null);
      try {
        const roles = await rbacRepository.getRoles();
        if (!mounted) return;

        const formattedRoles = roles
          .filter((item) =>
            ["member", "manager", "admin"].includes(item.id)
          )
          .map((item) => ({
            id: item.id as MemberRole,
            name: item.name,
            description: item.description,
            typicalUsers: item.typicalUsers,
          }));

        // Use formatted roles if available, otherwise use default fallback
        const rolesToUse = formattedRoles.length > 0 ? formattedRoles : DEFAULT_ROLES;
        setRoleOptions(rolesToUse);

        if (rolesToUse.length > 0) {
          const availableIds = rolesToUse.map((item) => item.id);
          setRole((prev) =>
            availableIds.includes(prev)
              ? prev
              : availableIds.includes("member")
                ? "member"
                : rolesToUse[0].id
          );
        }
      } catch (error) {
        console.error("[InviteMember] Failed to load RBAC roles", error);
        if (mounted) {
          // Use default roles on error for graceful degradation
          setRoleOptions(DEFAULT_ROLES);
          setRole("member");
          setRolesError(
            "Using default roles. Some features may be limited."
          );
        }
      } finally {
        if (mounted) {
          setIsLoadingRoles(false);
        }
      }
    };

    loadRoles();
    return () => {
      mounted = false;
    };
  }, []);

  const handleInvite = async () => {
    // Validate name
    if (!name.trim()) {
      toast({
        title: "Name required",
        description: "Please enter the member's name",
        variant: "destructive",
      });
      return;
    }

    // Validate email
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      toast({
        title: "Invalid email",
        description: "Please enter a valid email address",
        variant: "destructive",
      });
      return;
    }

    setIsLoading(true);
    setShowSuccess(false);
    try {
      // Use the onInvite prop (handles mutation and cache invalidation)
      await onInvite({
        email: email.trim(),
        name: name.trim(),
        role: role as MemberRole,
        sendEmail: true,
      });

      // Success - mutation completed
      setInvitedEmail(email);
      setShowSuccess(true);
      toast({
        title: "Invitation sent",
        description: `${email} has been invited to join your team`,
      });
      setEmail("");
      setName("");
      setRole((prev) => {
        if (roleOptions.some((option) => option.id === "member")) {
          return "member";
        }
        return roleOptions[0]?.id ?? prev;
      });

      // Hide success message after 5 seconds
      setTimeout(() => {
        setShowSuccess(false);
      }, 5000);
    } catch (error: any) {
      toast({
        title: "Error",
        description: error.message || "Failed to send invitation. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const currentRoleDetails = useMemo(
    () => roleOptions.find((option) => option.id === role),
    [roleOptions, role]
  );

  if (!canInvite) {
    return null;
  }

  return (
    <div className="space-y-6">
      {showSuccess && (
        <div className="flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 px-4 py-3">
          <CheckCircle2 className="h-5 w-5 flex-none text-green-600" />
          <div className="flex-1">
            <p className="text-sm font-medium text-green-900">Invitation sent successfully</p>
            <p className="text-xs text-green-700 mt-0.5">{invitedEmail} will receive an email shortly</p>
          </div>
        </div>
      )}

      {rolesError && (
        <div className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
          {rolesError}
        </div>
      )}

      <div className="space-y-5">
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="name" className="text-sm font-medium text-gray-900">
              Full Name
            </Label>
            <Input
              id="name"
              type="text"
              placeholder="John Doe"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="email" className="text-sm font-medium text-gray-900">
              Email Address
            </Label>
            <Input
              id="email"
              type="email"
              placeholder="colleague@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isLoading}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="role" className="text-sm font-medium text-gray-900">
            Role
          </Label>
          <Select
            value={role}
            onValueChange={(value) => setRole(value as MemberRole)}
            disabled={isLoading || isLoadingRoles || roleOptions.length === 0}
          >
            <SelectTrigger id="role" className="w-full justify-between text-left">
              <SelectValue placeholder={isLoadingRoles ? "Loading roles..." : "Select a role"} />
            </SelectTrigger>
            <SelectContent container={selectPortalContainer}>
              {roleOptions.map((option) => (
                <SelectItem key={option.id} value={option.id}>
                  <div className="space-y-1 text-left">
                    <p className="font-medium">{option.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {option.description}
                    </p>
                    {option.typicalUsers && (
                      <p className="text-xs text-muted-foreground">
                        Typical users: {option.typicalUsers}
                      </p>
                    )}
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {currentRoleDetails && (
            <p className="text-xs text-muted-foreground">
              {currentRoleDetails.description}
            </p>
          )}
        </div>

        <Button
          onClick={handleInvite}
          disabled={
            !email ||
            !name ||
            isLoading ||
            isLoadingRoles ||
            roleOptions.length === 0
          }
          className="w-full sm:w-auto"
        >
          {isLoading ? (
            "Sending invitation..."
          ) : (
            <>
              <Send className="mr-2 h-4 w-4" />
              Send Invitation
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
