"use client";

import { useEffect, useState, type FormEvent } from "react";
import toast from "react-hot-toast";
import {
  changeMerchantPassword,
  fetchMerchantMe,
  getMerchantProfile,
  updateMerchantProfile,
  updateStoredMerchantProfile,
  type MerchantProfile,
} from "@/lib/merchant-auth";
import { Button } from "@/components/ui/button";
import {
  Card,
  LoadingBlock,
  PageHeader,
} from "../../components/admin/ui";

export default function MerchantProfilePage() {
  const [profile, setProfile] = useState<MerchantProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [saving, setSaving] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [changingPw, setChangingPw] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      try {
        const me = await fetchMerchantMe();
        if (cancelled) return;
        setProfile(me);
        setName(me.name);
        setEmail(me.email);
        updateStoredMerchantProfile(me);
      } catch {
        const cached = getMerchantProfile();
        if (cached && !cancelled) {
          setProfile(cached);
          setName(cached.name);
          setEmail(cached.email);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleProfile(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const updated = await updateMerchantProfile(name.trim(), email.trim());
      setProfile(updated);
      updateStoredMerchantProfile(updated);
      toast.success("Profile updated.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update.");
    } finally {
      setSaving(false);
    }
  }

  async function handlePassword(e: FormEvent) {
    e.preventDefault();
    setChangingPw(true);
    try {
      await changeMerchantPassword(currentPassword, newPassword);
      setCurrentPassword("");
      setNewPassword("");
      toast.success("Password changed.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to change password.",
      );
    } finally {
      setChangingPw(false);
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        title="Profile"
        description="Your merchant account details."
      />

      {loading ? (
        <Card>
          <LoadingBlock />
        </Card>
      ) : (
        <>
          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Account
            </h3>
            <form
              onSubmit={(e) => void handleProfile(e)}
              className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2"
            >
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Name
                </span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Email
                </span>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              {profile?.business_name && (
                <p className="sm:col-span-2 text-[12.5px] text-[#8a97a8]">
                  Business:{" "}
                  <span className="font-medium text-[#3d4b5c]">
                    {profile.business_name}
                  </span>
                </p>
              )}
              <div className="sm:col-span-2">
                <Button
                  type="submit"
                  disabled={saving}
                  className="h-10 rounded-full bg-[#06163a] px-5 text-[13px] font-semibold text-white shadow-none hover:bg-[#0b2048]"
                >
                  {saving ? "Saving..." : "Save profile"}
                </Button>
              </div>
            </form>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="text-[15px] font-semibold text-[#1f2a37]">
              Change password
            </h3>
            <form
              onSubmit={(e) => void handlePassword(e)}
              className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2"
            >
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  Current password
                </span>
                <input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-[11px] font-semibold uppercase tracking-wide text-[#8a97a8]">
                  New password
                </span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  minLength={8}
                  className="h-10 rounded-lg border border-[#e8eef4] px-3 text-[13px] outline-none focus:border-[#3b9eff] focus:ring-2 focus:ring-[#3b9eff]/15"
                  required
                />
              </label>
              <div className="sm:col-span-2">
                <Button
                  type="submit"
                  disabled={changingPw}
                  className="h-10 rounded-full bg-[#ff5e16] px-5 text-[13px] font-semibold text-white shadow-none hover:bg-[#e8530f]"
                >
                  {changingPw ? "Updating..." : "Update password"}
                </Button>
              </div>
            </form>
          </Card>
        </>
      )}
    </div>
  );
}
