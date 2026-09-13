import { LandingNavbar } from "@/components/landing/navbar";
import { Hero } from "@/components/landing/hero";
import {
  ArchitectureSection,
  FeaturesSection,
  HowItWorksSection,
  ProblemSection,
  UseCasesSection,
} from "@/components/landing/sections";
import {
  DashboardPreviewSection,
  FaqSection,
  FinalCtaSection,
  LandingFooter,
  PricingSection,
} from "@/components/landing/pricing-faq-footer";

export default function LandingPage() {
  return (
    <div className="font-(--font-lp-body) flex min-h-screen flex-col">
      <LandingNavbar />
      <main className="flex-1">
        <Hero />
        <ProblemSection />
        <HowItWorksSection />
        <FeaturesSection />
        <ArchitectureSection />
        <DashboardPreviewSection />
        <UseCasesSection />
        <PricingSection />
        <FaqSection />
        <FinalCtaSection />
      </main>
      <LandingFooter />
    </div>
  );
}
