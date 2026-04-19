"use client";

export type ProfileFormData = {
  full_name: string;
  company: string;
  referral_source: string;
  phone: string;
  job_function: string;
  country: string;
  avatar_url: string;
  bio: string;
};

export type ProfileViewData = {
  id: string;
  username: string;
  email: string;
  status: string;
  on_boarding: boolean;
  profile: ProfileFormData;
};
