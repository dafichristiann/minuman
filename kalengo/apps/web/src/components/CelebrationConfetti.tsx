import React, { useMemo } from "react";

export interface CelebrationConfettiProps {
  active?: boolean;
}

interface Particle {
  id: number;
  left: number;
  size: number;
  duration: number;
  delay: number;
  rotate: number;
  type: "can" | "circle" | "square" | "star" | "ribbon";
  color: string;
}

const COLORS = ["#ffd23f", "#ff4b3e", "#3ddc97", "#2ec4f1", "#d6336c", "#1b1b2f"];
const TYPES: Array<Particle["type"]> = ["can", "circle", "square", "star", "ribbon"];

export const CelebrationConfetti: React.FC<CelebrationConfettiProps> = ({ active = true }) => {
  const particles = useMemo<Particle[]>(() => {
    return Array.from({ length: 32 }, (_, i) => ({
      id: i,
      left: Math.round((i * 100) / 32 + (Math.sin(i * 3.7) * 8)),
      size: Math.round(14 + (i % 5) * 4),
      duration: Number((2.2 + (i % 4) * 0.45).toFixed(2)),
      delay: Number(((i % 8) * 0.18).toFixed(2)),
      rotate: Math.round((i * 45) % 360),
      type: TYPES[i % TYPES.length] ?? "circle",
      color: COLORS[i % COLORS.length] ?? "#ffd23f",
    }));
  }, []);

  if (!active) return null;

  return (
    <div
      className="celebration-confetti-layer"
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        overflow: "hidden",
        pointerEvents: "none",
        zIndex: 10,
      }}
    >
      {particles.map((p) => {
        let content: React.ReactNode = null;
        if (p.type === "can") {
          content = "🥫";
        } else if (p.type === "star") {
          content = "⭐";
        } else if (p.type === "circle") {
          content = (
            <span
              style={{
                display: "inline-block",
                width: `${p.size}px`,
                height: `${p.size}px`,
                borderRadius: "50%",
                backgroundColor: p.color,
                border: "2px solid #1b1b2f",
              }}
            />
          );
        } else if (p.type === "square") {
          content = (
            <span
              style={{
                display: "inline-block",
                width: `${p.size}px`,
                height: `${p.size}px`,
                backgroundColor: p.color,
                border: "2px solid #1b1b2f",
                transform: `rotate(${p.rotate}deg)`,
              }}
            />
          );
        } else {
          // ribbon
          content = (
            <span
              style={{
                display: "inline-block",
                width: `${p.size * 1.5}px`,
                height: `${p.size * 0.45}px`,
                borderRadius: "3px",
                backgroundColor: p.color,
                border: "1.5px solid #1b1b2f",
                transform: `rotate(${p.rotate}deg)`,
              }}
            />
          );
        }

        return (
          <div
            key={p.id}
            className="confetti-particle"
            style={{
              position: "absolute",
              left: `${Math.max(2, Math.min(96, p.left))}%`,
              top: "-40px",
              fontSize: `${p.size}px`,
              lineHeight: 1,
              animationDuration: `${p.duration}s`,
              animationDelay: `${p.delay}s`,
            }}
          >
            {content}
          </div>
        );
      })}
    </div>
  );
};
