import { LandingNavbar } from "@/components/landing/navbar";
import { Hero } from "@/components/landing/hero";
import { Reveal } from "@/components/landing/reveal";
import { HighlightsSection } from "@/components/landing/highlights";
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
        <Reveal>
          <ProblemSection />
        </Reveal>
        <Reveal>
          <HowItWorksSection />
        </Reveal>
        <Reveal>
          <FeaturesSection />
        </Reveal>
        <Reveal>
          <ArchitectureSection />
        </Reveal>
        <Reveal>
          <DashboardPreviewSection />
        </Reveal>
        <Reveal>
          <UseCasesSection />
        </Reveal>
        <Reveal>
          <PricingSection />
        </Reveal>
        <Reveal>
          <FaqSection />
        </Reveal>
        <Reveal>
          <HighlightsSection />
        </Reveal>
        <Reveal>
          <FinalCtaSection />
        </Reveal>
      </main>
      <LandingFooter />
    </div>
  );
}
