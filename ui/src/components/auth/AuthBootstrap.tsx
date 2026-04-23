"use client";

import { useEffect } from "react";
import { bootstrapSession, installAuthFetchInterceptor } from "./auth-session";

export default function AuthBootstrap() {
  useEffect(() => {
    installAuthFetchInterceptor();
    void bootstrapSession({ redirectOnFailure: true });
  }, []);

  return null;
}
