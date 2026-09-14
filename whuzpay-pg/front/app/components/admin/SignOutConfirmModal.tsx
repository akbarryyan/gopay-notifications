"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

type SignOutConfirmModalProps = {
  open: boolean;
  loading?: boolean;
  onClose: () => void;
  onConfirm: () => void;
};

export default function SignOutConfirmModal({
  open,
  loading = false,
  onClose,
  onConfirm,
}: SignOutConfirmModalProps) {
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !loading) onClose();
      }}
    >
      <AlertDialogContent
        size="default"
        className="max-w-sm border border-[#e8eef4] bg-white p-0 shadow-[0_24px_60px_rgba(16,38,73,0.18)] sm:max-w-sm"
      >
        <AlertDialogHeader className="place-items-center gap-0 p-6 pb-0 text-center sm:place-items-center sm:text-center">
          <AlertDialogMedia className="mb-3 size-12 rounded-full bg-[#fff1ed] text-[#e85d3b]">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.75}
              className="size-6"
              aria-hidden
            >
              <path
                d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <path
                d="M16 17l5-5-5-5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
              <path d="M21 12H9" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </AlertDialogMedia>
          <AlertDialogTitle className="text-[18px] font-semibold tracking-tight text-[#1f2a37]">
            Sign out?
          </AlertDialogTitle>
          <AlertDialogDescription className="mt-2 text-[13.5px] leading-relaxed text-[#8a97a8]">
            You will need to sign in again to access the admin panel. Any unsaved
            changes may be lost.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter className="mx-0 mb-0 grid grid-cols-2 gap-3 rounded-b-xl border-t border-[#eef2f6] bg-[#fafbfc] p-4 sm:flex-row">
          <AlertDialogCancel
            disabled={loading}
            className="h-10 rounded-lg border border-[#e8eef4] bg-white text-[13.5px] font-semibold text-[#3d4b5c] shadow-none hover:bg-[#f4f7fb]"
          >
            Cancel
          </AlertDialogCancel>
          {/* Use default (not destructive): shadcn destructive is bg-destructive/10 translucent. */}
          <AlertDialogAction
            disabled={loading}
            variant="default"
            className="h-10 rounded-lg border-transparent bg-[#e85d3b]! text-[13.5px] font-semibold text-white! shadow-[0_4px_14px_rgba(232,93,59,0.28)] hover:bg-[#d64f30]! hover:text-white! focus-visible:border-[#e85d3b] focus-visible:ring-[#e85d3b]/30 disabled:bg-[#e85d3b]/70! disabled:text-white!"
            onClick={(e) => {
              e.preventDefault();
              onConfirm();
            }}
          >
            {loading ? (
              <span className="inline-flex items-center gap-2">
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                Signing out...
              </span>
            ) : (
              "Sign out"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
