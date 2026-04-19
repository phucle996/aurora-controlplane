"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { getActor } from "@/components/auth/auth-session";
import UserAddressCard from "./UserAddressCard";
import UserInfoCard from "./UserInfoCard";
import UserMetaCard from "./UserMetaCard";
import type { ProfileFormData, ProfileViewData } from "./types";

const emptyProfile: ProfileFormData = {
  full_name: "",
  company: "",
  referral_source: "",
  phone: "",
  job_function: "",
  country: "",
  avatar_url: "",
  bio: "",
};

export default function ProfileClient() {
  const [profile, setProfile] = useState<ProfileViewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [savingSection, setSavingSection] = useState("");

  const fetchProfile = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const actor = getActor();
      if (actor == null || actor.username.trim() === "") {
        throw new Error("Your session details are not available right now.");
      }

      setProfile({
        id: actor.user_id,
        username: actor.username,
        email: actor.email,
        status: "",
        on_boarding: false,
        profile: {
          ...emptyProfile,
          full_name: actor.full_name ?? "",
        },
      });
    } catch (fetchError) {
      setError(
        fetchError instanceof Error
          ? fetchError.message
          : "Unable to load your profile right now.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchProfile();
  }, [fetchProfile]);

  const updateProfile = useCallback(
    async (section: string, _patch: Partial<ProfileFormData>) => {
      if (profile == null) {
        return;
      }

      setSavingSection(section);
      setError("");
      setSuccess("");

      try {
        throw new Error("Profile update API is not available in the new backend yet.");
      } catch (saveError) {
        setError(
          saveError instanceof Error
            ? saveError.message
            : "Unable to update your profile right now.",
        );
        throw saveError;
      } finally {
        setSavingSection("");
      }
    },
    [profile],
  );

  const content = useMemo(() => {
    if (loading) {
      return (
        <div className="rounded-2xl border border-dashed border-gray-300 px-6 py-12 text-center text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
          Loading your profile...
        </div>
      );
    }

    if (profile == null) {
      return (
        <div className="rounded-2xl border border-error-200 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300">
          {error || "Unable to load your profile right now."}
        </div>
      );
    }

    return (
      <div className="space-y-6">
        <UserMetaCard
          fullName={profile.profile.full_name}
          username={profile.username}
          email={profile.email}
          avatarURL={profile.profile.avatar_url}
          jobFunction={profile.profile.job_function}
          country={profile.profile.country}
          company={profile.profile.company}
          saving={savingSection === "meta"}
          onSave={(patch) => updateProfile("meta", patch)}
        />
        <UserInfoCard
          fullName={profile.profile.full_name}
          username={profile.username}
          email={profile.email}
          phone={profile.profile.phone}
          bio={profile.profile.bio}
          saving={savingSection === "info"}
          onSave={(patch) => updateProfile("info", patch)}
        />
        <UserAddressCard
          company={profile.profile.company}
          jobFunction={profile.profile.job_function}
          country={profile.profile.country}
          referralSource={profile.profile.referral_source}
          saving={savingSection === "work"}
          onSave={(patch) => updateProfile("work", patch)}
        />
      </div>
    );
  }, [error, loading, profile, savingSection, updateProfile]);

  return (
    <div className="space-y-6">
      {(error !== "" || success !== "") && profile != null && (
        <div
          className={`rounded-2xl px-4 py-3 text-sm ${
            error !== ""
              ? "border border-error-200 bg-error-50 text-error-700 dark:border-error-500/30 dark:bg-error-500/10 dark:text-error-300"
              : "border border-success-200 bg-success-50 text-success-700 dark:border-success-500/30 dark:bg-success-500/10 dark:text-success-300"
          }`}
        >
          {error !== "" ? error : success}
        </div>
      )}
      {content}
    </div>
  );
}
