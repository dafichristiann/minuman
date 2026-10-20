import { render } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { CelebrationConfetti } from "./CelebrationConfetti";

describe("CelebrationConfetti", () => {
  test("merender layer partikel konfeti saat active true", () => {
    const { container } = render(<CelebrationConfetti active={true} />);

    const layer = container.querySelector(".celebration-confetti-layer");
    expect(layer).toBeInTheDocument();
    expect(layer).toHaveAttribute("aria-hidden", "true");

    const particles = container.querySelectorAll(".confetti-particle");
    expect(particles.length).toBeGreaterThan(0);
  });

  test("tidak merender apa-apa saat active false", () => {
    const { container } = render(<CelebrationConfetti active={false} />);
    expect(container.firstChild).toBeNull();
  });
});
