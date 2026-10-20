import React from "react";

export interface MascotProps extends React.SVGProps<SVGSVGElement> {
  size?: number | string;
  className?: string;
  title?: string;
  alt?: string;
}

/**
 * 1. Kalengo Soda Lemon (can-soda-lemon)
 * Yellow can mascot with winking pie-eye, rubber-hose white glove hands, lemon slice accessory.
 */
export function CanSodaLemon({
  size = 180,
  className = "",
  title = "Maskot Kalengo Soda Lemon",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--lemon ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow under feet */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Rubber hose legs & cartoon boots */}
        <path d="M 68 162 Q 62 186 64 198" fill="none" />
        <path d="M 112 162 Q 118 186 116 198" fill="none" />
        {/* Left boot */}
        <path
          d="M 52 204 C 52 196 60 195 65 195 C 72 195 78 198 78 205 C 78 208 72 210 63 210 C 56 210 52 208 52 204 Z"
          fill="#1b1b2f"
        />
        {/* Right boot */}
        <path
          d="M 102 205 C 102 198 108 195 115 195 C 120 195 128 196 128 204 C 128 208 124 210 117 210 C 108 210 102 208 102 205 Z"
          fill="#1b1b2f"
        />

        {/* Rubber hose left arm - waving */}
        <path d="M 45 105 Q 22 92 24 72" fill="none" strokeWidth="3.5" />
        {/* Left cartoon glove (3 fingers + cuff) */}
        <ellipse cx="25" cy="71" rx="9" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          <path d="M 17 68 C 14 62 19 53 25 54 C 29 55 31 60 30 66 C 34 60 41 62 40 68 C 39 73 34 76 28 76 C 21 76 17 73 17 68 Z" />
          {/* Glove dart lines */}
          <line x1="22" y1="62" x2="25" y2="67" />
          <line x1="27" y1="63" x2="29" y2="68" />
        </g>

        {/* Rubber hose right arm - hand on hip */}
        <path d="M 135 105 Q 155 115 152 135" fill="none" strokeWidth="3.5" />
        {/* Right cartoon glove */}
        <ellipse cx="151" cy="135" rx="7" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="150" cy="144" r="9" />
          <path d="M 143 140 C 141 146 148 152 153 149" fill="none" />
        </g>

        {/* Pull tab "tuft" on top */}
        <path
          d="M 85 36 C 85 24 95 24 95 36 Z"
          fill="#ffd23f"
        />
        <circle cx="90" cy="30" r="3" fill="#1b1b2f" stroke="none" />

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#ffe9bd" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#ffd23f" />

        {/* Can body */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#ffd23f"
        />

        {/* Can bottom rim curve */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#ffe9bd" />

        {/* White graphic label belly sash */}
        <path
          d="M 45 84 Q 90 95 135 84 L 135 132 Q 90 143 45 132 Z"
          fill="#ffffff"
        />

        {/* Lemon Slice Accessory on Top Rim */}
        <g transform="translate(118, 30) rotate(15)">
          <circle cx="14" cy="14" r="14" fill="#ff4b3e" />
          <circle cx="14" cy="14" r="12" fill="#ffffff" />
          <circle cx="14" cy="14" r="10" fill="#ffd23f" />
          {/* Segments */}
          <path d="M 14 4 L 14 24 M 4 14 L 24 14 M 7 7 L 21 21 M 7 21 L 21 7" stroke="#ffffff" strokeWidth="2" />
        </g>

        {/* Bubbles on sash */}
        <circle cx="60" cy="100" r="3" fill="#ffd23f" stroke="none" />
        <circle cx="120" cy="116" r="4" fill="#ffd23f" stroke="none" />
        <circle cx="126" cy="98" r="2" fill="#ffd23f" stroke="none" />

        {/* Cheeks blush */}
        <circle cx="62" cy="118" r="6" fill="#ff4b3e" opacity="0.8" stroke="none" />
        <circle cx="118" cy="118" r="6" fill="#ff4b3e" opacity="0.8" stroke="none" />

        {/* Left eye: 1930s pie-cut cartoon eye */}
        <g fill="#1b1b2f">
          <ellipse cx="72" cy="105" rx="7" ry="10" stroke="none" />
          {/* Pie cutout */}
          <polygon points="72,105 66,101 67,109" fill="#ffffff" stroke="none" />
          {/* Eyebrow */}
          <path d="M 65 91 Q 72 87 79 91" fill="none" strokeWidth="3" />
        </g>

        {/* Right eye: Playful Winking curve */}
        <path d="M 103 105 Q 112 113 121 104" fill="none" strokeWidth="3.5" />
        <line x1="112" y1="110" x2="114" y2="115" strokeWidth="3" />
        {/* Right eyebrow raised */}
        <path d="M 104 88 Q 112 84 120 88" fill="none" strokeWidth="3" />

        {/* Cartoon smiling mouth with tongue */}
        <path
          d="M 77 121 Q 90 137 103 121 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 83 128 Q 90 124 97 128 Q 90 136 83 128 Z"
          fill="#ff4b3e"
          stroke="none"
        />
      </g>
    </svg>
  );
}

/**
 * 2. Kalengo Kopi Susu (can-kopi-susu)
 * Warm coffee brown can mascot with sleepy/chilled smiling face, coffee bean bowtie.
 */
export function CanKopiSusu({
  size = 180,
  className = "",
  title = "Maskot Kalengo Kopi Susu",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--kopi ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow under feet */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Sleepy steam curls */}
        <path d="M 85 24 Q 82 15 88 10 Q 94 5 90 0" fill="none" stroke="#6f432a" strokeWidth="2.5" opacity="0.7" />
        <path d="M 97 22 Q 102 14 98 8" fill="none" stroke="#6f432a" strokeWidth="2" opacity="0.5" />

        {/* Legs & comfy loafers */}
        <path d="M 70 162 Q 68 184 68 198" fill="none" />
        <path d="M 110 162 Q 112 184 112 198" fill="none" />
        <path
          d="M 54 204 C 54 197 62 196 68 196 C 75 196 80 199 80 205 C 80 209 74 210 65 210 C 58 210 54 208 54 204 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 100 205 C 100 199 105 196 112 196 C 118 196 126 197 126 204 C 126 208 122 210 115 210 C 106 210 100 208 100 205 Z"
          fill="#1b1b2f"
        />

        {/* Rubber hose arms resting relaxed on tummy */}
        <path d="M 45 106 Q 30 120 48 136" fill="none" strokeWidth="3.5" />
        <ellipse cx="49" cy="137" rx="6" ry="4" fill="#1b1b2f" />
        <circle cx="56" cy="141" r="8" fill="#ffffff" />

        <path d="M 135 106 Q 150 120 132 136" fill="none" strokeWidth="3.5" />
        <ellipse cx="131" cy="137" rx="6" ry="4" fill="#1b1b2f" />
        <circle cx="124" cy="141" r="8" fill="#ffffff" />

        {/* Pull tab */}
        <path d="M 85 36 C 85 24 95 24 95 36 Z" fill="#c49a6c" />
        <circle cx="90" cy="30" r="3" fill="#1b1b2f" stroke="none" />

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#e6ccb2" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#8d5b4c" />

        {/* Can body - Coffee rich brown */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#7f4f3b"
        />

        {/* Creamy milk froth swirl layer */}
        <path
          d="M 45 64 Q 68 74 90 64 Q 112 55 135 66 L 135 46 L 45 46 Z"
          fill="#ede0d4"
        />

        {/* Bottom rim */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#653d2d" />

        {/* Coffee Bean Bowtie */}
        <g transform="translate(90, 72)">
          {/* Left bean lobe */}
          <path
            d="M -2 0 C -12 -9 -22 0 -14 10 C -6 16 -1 5 -2 0 Z"
            fill="#3d2314"
          />
          <path d="M -14 2 Q -8 5 -3 2" fill="none" stroke="#e6ccb2" strokeWidth="1.5" />
          {/* Right bean lobe */}
          <path
            d="M 2 0 C 12 -9 22 0 14 10 C 6 16 1 5 2 0 Z"
            fill="#3d2314"
          />
          <path d="M 14 2 Q 8 5 3 2" fill="none" stroke="#e6ccb2" strokeWidth="1.5" />
          {/* Knot */}
          <circle cx="0" cy="4" r="4" fill="#ffd23f" />
        </g>

        {/* Warm blush */}
        <ellipse cx="64" cy="116" rx="7" ry="4" fill="#e29578" opacity="0.8" stroke="none" />
        <ellipse cx="116" cy="116" rx="7" ry="4" fill="#e29578" opacity="0.8" stroke="none" />

        {/* Sleepy/chilled eyes (curved happy sleep arcs) */}
        <path d="M 64 106 Q 74 115 82 106" fill="none" strokeWidth="3.5" />
        <path d="M 98 106 Q 106 115 116 106" fill="none" strokeWidth="3.5" />
        {/* Soft sleepy eyelids */}
        <path d="M 66 101 Q 74 98 81 101" fill="none" strokeWidth="2.5" />
        <path d="M 99 101 Q 106 98 114 101" fill="none" strokeWidth="2.5" />

        {/* Calm sweet smile */}
        <path d="M 83 124 Q 90 131 97 124" fill="none" strokeWidth="3.5" />
      </g>
    </svg>
  );
}

/**
 * 3. Kalengo Teh Melati (can-teh-melati)
 * Green/mint can mascot with cheerful wide smile, white jasmine flower accessory.
 */
export function CanTehMelati({
  size = 180,
  className = "",
  title = "Maskot Kalengo Teh Melati",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--teh ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Happy little bounce legs */}
        <path d="M 68 162 Q 62 184 60 198" fill="none" />
        <path d="M 112 162 Q 118 184 120 198" fill="none" />
        <path
          d="M 48 204 C 48 197 56 195 62 195 C 69 195 74 198 74 205 C 74 209 68 210 59 210 C 52 210 48 208 48 204 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 106 205 C 106 198 111 195 118 195 C 124 195 132 197 132 204 C 132 208 128 210 121 210 C 112 210 106 208 106 205 Z"
          fill="#1b1b2f"
        />

        {/* Cheering arms raised up high! */}
        <path d="M 45 100 Q 20 80 18 56" fill="none" strokeWidth="3.5" />
        <ellipse cx="19" cy="56" rx="8" ry="4" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="18" cy="46" r="10" />
          <path d="M 14 42 C 16 36 24 38 23 46" fill="none" />
        </g>

        <path d="M 135 100 Q 160 80 162 56" fill="none" strokeWidth="3.5" />
        <ellipse cx="161" cy="56" rx="8" ry="4" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="162" cy="46" r="10" />
          <path d="M 166 42 C 164 36 156 38 157 46" fill="none" />
        </g>

        {/* Pull tab */}
        <path d="M 85 36 C 85 24 95 24 95 36 Z" fill="#a7f3d0" />
        <circle cx="90" cy="30" r="3" fill="#1b1b2f" stroke="none" />

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#d1fae5" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#3ddc97" />

        {/* Can body - Mint Green */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#3ddc97"
        />

        {/* Bottom rim */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#2eb87c" />

        {/* Jasmine Flower accessory pinned to top rim */}
        <g transform="translate(122, 28)">
          {/* Flower petals */}
          <path d="M 14 0 C 18 6 18 12 14 14 C 10 12 10 6 14 0 Z" fill="#ffffff" />
          <path d="M 28 14 C 22 18 16 18 14 14 C 16 10 22 10 28 14 Z" fill="#ffffff" />
          <path d="M 14 28 C 10 22 10 16 14 14 C 18 16 18 22 14 28 Z" fill="#ffffff" />
          <path d="M 0 14 C 6 10 12 10 14 14 C 12 18 6 18 0 14 Z" fill="#ffffff" />
          <circle cx="14" cy="14" r="4" fill="#ffd23f" />
        </g>

        {/* Tea leaf motif on belly */}
        <path
          d="M 90 70 Q 102 78 98 90 Q 86 86 90 70 Z"
          fill="#208b5c"
          opacity="0.25"
          stroke="none"
        />

        {/* Rosy cheeks */}
        <circle cx="62" cy="116" r="6" fill="#ff4b3e" opacity="0.6" stroke="none" />
        <circle cx="118" cy="116" r="6" fill="#ff4b3e" opacity="0.6" stroke="none" />

        {/* Cheerful wide-open pie eyes */}
        <g fill="#1b1b2f">
          <ellipse cx="71" cy="102" rx="7" ry="10" stroke="none" />
          <polygon points="71,102 65,98 66,106" fill="#ffffff" stroke="none" />
          <path d="M 64 88 Q 71 84 78 88" fill="none" strokeWidth="3" />

          <ellipse cx="109" cy="102" rx="7" ry="10" stroke="none" />
          <polygon points="109,102 115,98 114,106" fill="#ffffff" stroke="none" />
          <path d="M 102 88 Q 109 84 116 88" fill="none" strokeWidth="3" />
        </g>

        {/* Cheerful wide open smile */}
        <path
          d="M 72 118 Q 90 146 108 118 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 80 127 Q 90 122 100 127 Q 90 142 80 127 Z"
          fill="#ff4b3e"
          stroke="none"
        />
      </g>
    </svg>
  );
}

/**
 * 4. Kalengo Lite Soda Zero (can-lite-zero)
 * Light blue can mascot with retro sunglasses and cool pose.
 */
export function CanLiteZero({
  size = 180,
  className = "",
  title = "Maskot Kalengo Lite Zero",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--zero ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Relaxed crossed legs */}
        <path d="M 70 162 Q 74 184 76 198" fill="none" />
        <path d="M 110 162 Q 104 182 106 198" fill="none" />
        <path
          d="M 64 204 C 64 197 72 195 78 195 C 85 195 90 198 90 205 C 90 209 84 210 75 210 C 68 210 64 208 64 204 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 98 205 C 98 198 103 195 110 195 C 116 195 124 197 124 204 C 124 208 120 210 113 210 C 104 210 98 208 98 205 Z"
          fill="#1b1b2f"
        />

        {/* Left arm: Cool thumb-up */}
        <path d="M 45 106 Q 20 115 25 132" fill="none" strokeWidth="3.5" />
        <ellipse cx="27" cy="132" rx="7" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          {/* Fist + thumb up */}
          <circle cx="28" cy="140" r="9" />
          <path d="M 28 131 L 28 123 C 28 120 33 120 33 124 L 33 133" />
        </g>

        {/* Right arm: On hip */}
        <path d="M 135 106 Q 158 116 150 135" fill="none" strokeWidth="3.5" />
        <ellipse cx="149" cy="135" rx="7" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="146" cy="144" r="8" />
        </g>

        {/* Pull tab angled back like hair */}
        <path d="M 85 36 C 82 22 92 20 98 32 Z" fill="#b9e7fe" />
        <circle cx="91" cy="27" r="3" fill="#1b1b2f" stroke="none" />

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#cbebfd" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#2ec4f1" />

        {/* Can body - Sky Blue */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#2ec4f1"
        />

        {/* Bold Diagonal ZERO ribbon */}
        <path
          d="M 45 74 L 135 94 L 135 110 L 45 90 Z"
          fill="#ffffff"
        />
        <text
          x="90"
          y="105"
          fill="#1b1b2f"
          fontFamily="system-ui, sans-serif"
          fontWeight="900"
          fontSize="14"
          letterSpacing="3"
          textAnchor="middle"
          transform="rotate(12.5 90 102)"
          stroke="none"
        >
          ZERO
        </text>

        {/* Bottom rim */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#1aa8d4" />

        {/* Cool Retro Sunglasses */}
        <g>
          {/* Bridge */}
          <rect x="85" y="86" width="10" height="4" fill="#1b1b2f" />
          {/* Left lens */}
          <path
            d="M 52 83 L 85 83 C 86 98 80 106 66 106 C 54 106 50 97 52 83 Z"
            fill="#1b1b2f"
          />
          {/* Left shine streaks */}
          <line x1="58" y1="87" x2="66" y2="102" stroke="#ffffff" strokeWidth="2.5" />
          <line x1="64" y1="87" x2="69" y2="98" stroke="#ffffff" strokeWidth="1.5" />

          {/* Right lens */}
          <path
            d="M 95 83 L 128 83 C 130 97 126 106 114 106 C 100 106 94 98 95 83 Z"
            fill="#1b1b2f"
          />
          {/* Right shine streaks */}
          <line x1="101" y1="87" x2="109" y2="102" stroke="#ffffff" strokeWidth="2.5" />
          <line x1="107" y1="87" x2="112" y2="98" stroke="#ffffff" strokeWidth="1.5" />
        </g>

        {/* Confident side smirk */}
        <path d="M 88 126 Q 98 132 106 122" fill="none" strokeWidth="3.5" />
        {/* Dimple line */}
        <path d="M 107 120 Q 109 124 106 126" fill="none" strokeWidth="2.5" />
      </g>
    </svg>
  );
}

/**
 * 5. Kalengo Buah Mangga (can-buah-mangga)
 * Bright orange/mango can mascot with laughing tongue-out face and mango leaf.
 */
export function CanBuahMangga({
  size = 180,
  className = "",
  title = "Maskot Kalengo Rasa Mangga",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--mangga ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Playful jumpy legs */}
        <path d="M 66 162 Q 58 184 56 198" fill="none" />
        <path d="M 114 162 Q 122 184 124 198" fill="none" />
        <path
          d="M 44 204 C 44 197 52 195 58 195 C 65 195 70 198 70 205 C 70 209 64 210 55 210 C 48 210 44 208 44 204 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 112 205 C 112 198 117 195 124 195 C 130 195 138 197 138 204 C 138 208 134 210 127 210 C 118 210 112 208 112 205 Z"
          fill="#1b1b2f"
        />

        {/* Arms wide open with playful jazz hands */}
        <path d="M 45 106 Q 16 110 18 90" fill="none" strokeWidth="3.5" />
        <ellipse cx="19" cy="90" rx="8" ry="4" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="18" cy="80" r="9" />
          <path d="M 14 74 L 14 70" />
          <path d="M 19 72 L 20 68" />
        </g>

        <path d="M 135 106 Q 164 110 162 90" fill="none" strokeWidth="3.5" />
        <ellipse cx="161" cy="90" rx="8" ry="4" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="162" cy="80" r="9" />
          <path d="M 166 74 L 166 70" />
          <path d="M 161 72 L 160 68" />
        </g>

        {/* Pull tab */}
        <path d="M 85 36 C 85 24 95 24 95 36 Z" fill="#ffd166" />
        <circle cx="90" cy="30" r="3" fill="#1b1b2f" stroke="none" />

        {/* Mango Leaf sprig on top */}
        <g transform="translate(98, 20) rotate(25)">
          <path
            d="M 0 16 C 6 6 18 0 28 2 C 24 12 14 20 0 16 Z"
            fill="#3ddc97"
          />
          <path d="M 0 16 Q 14 10 26 3" fill="none" stroke="#1b1b2f" strokeWidth="2" />
        </g>

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#ffe3a8" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#ff922b" />

        {/* Can body - Mango bright orange */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#ff922b"
        />

        {/* Mango blush gradient highlight */}
        <path
          d="M 45 90 Q 75 120 135 90 L 135 140 Q 75 165 45 140 Z"
          fill="#ffd23f"
          opacity="0.5"
          stroke="none"
        />

        {/* Bottom rim */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#e8590c" />

        {/* Big rosy blushing cheeks */}
        <circle cx="62" cy="116" r="7" fill="#ff4b3e" opacity="0.8" stroke="none" />
        <circle cx="118" cy="116" r="7" fill="#ff4b3e" opacity="0.8" stroke="none" />

        {/* Laughing happy squeezed eyes (^ ^) */}
        <path d="M 64 102 Q 72 90 80 102" fill="none" strokeWidth="4" />
        <path d="M 100 102 Q 108 90 116 102" fill="none" strokeWidth="4" />

        {/* Playful wide open laughing mouth with protruding tongue */}
        <path
          d="M 72 114 Q 90 136 108 114 Z"
          fill="#1b1b2f"
        />
        {/* Tongue sticking out (blewek) */}
        <path
          d="M 82 124 C 82 136 86 142 90 142 C 94 142 98 136 98 124 Z"
          fill="#ff4b3e"
        />
        <line x1="90" y1="126" x2="90" y2="137" stroke="#1b1b2f" strokeWidth="2" />
      </g>
    </svg>
  );
}

/**
 * 6. Kalengo Soda Jahe (can-soda-jahe)
 * Warm amber/gold can mascot with fiery energetic expression and flame sparks.
 */
export function CanSodaJahe({
  size = 180,
  className = "",
  title = "Maskot Kalengo Soda Jahe",
  alt,
  ...props
}: MascotProps) {
  const accessibleLabel = alt || title;
  return (
    <svg
      viewBox="0 0 180 220"
      width={size}
      height={size}
      className={`kalengo-mascot kalengo-mascot--jahe ${className}`}
      role="img"
      aria-label={accessibleLabel}
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <title>{accessibleLabel}</title>
      <g stroke="#1b1b2f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        {/* Shadow */}
        <ellipse cx="90" cy="210" rx="46" ry="7" fill="#1b1b2f" opacity="0.18" stroke="none" />

        {/* Fiery energetic sparks */}
        <g stroke="#ff4b3e" strokeWidth="2.5" fill="none">
          <path d="M 32 60 L 36 52 L 44 56" />
          <path d="M 148 56 L 154 50 L 158 58" />
        </g>
        <circle cx="36" cy="72" r="2.5" fill="#ffd23f" stroke="none" />
        <circle cx="150" cy="70" r="2.5" fill="#ffd23f" stroke="none" />

        {/* Power stance dynamic legs */}
        <path d="M 64 162 Q 54 186 52 198" fill="none" />
        <path d="M 116 162 Q 126 186 128 198" fill="none" />
        <path
          d="M 40 204 C 40 197 48 195 54 195 C 61 195 66 198 66 205 C 66 209 60 210 51 210 C 44 210 40 208 40 204 Z"
          fill="#1b1b2f"
        />
        <path
          d="M 116 205 C 116 198 121 195 128 195 C 134 195 142 197 142 204 C 142 208 138 210 131 210 C 122 210 116 208 116 205 Z"
          fill="#1b1b2f"
        />

        {/* Left arm: Power punch raised */}
        <path d="M 45 106 Q 18 90 26 66" fill="none" strokeWidth="3.5" />
        <ellipse cx="26" cy="66" rx="7" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          {/* Clenched power fist */}
          <circle cx="28" cy="58" r="9" />
          <path d="M 23 58 L 33 58" />
          <path d="M 24 63 L 32 63" />
        </g>

        {/* Right arm: Flexed bicep */}
        <path d="M 135 106 Q 162 98 152 76" fill="none" strokeWidth="3.5" />
        <ellipse cx="152" cy="76" rx="7" ry="5" fill="#1b1b2f" />
        <g fill="#ffffff">
          <circle cx="150" cy="68" r="9" />
          <path d="M 145 68 L 155 68" />
        </g>

        {/* Fiery flame pull tab */}
        <path
          d="M 85 36 Q 80 18 90 10 Q 94 22 100 24 Q 96 34 95 36 Z"
          fill="#ff4b3e"
        />
        <path
          d="M 88 32 Q 86 22 91 16 Q 93 23 96 26 Z"
          fill="#ffd23f"
          stroke="none"
        />

        {/* Can Lid / Top rim */}
        <ellipse cx="90" cy="46" rx="45" ry="11" fill="#fed7aa" />
        <ellipse cx="90" cy="45" rx="39" ry="8" fill="#f59f00" />

        {/* Can body - Amber ginger gold */}
        <path
          d="M 45 46 L 45 152 C 45 166 135 166 135 152 L 135 46"
          fill="#f59f00"
        />

        {/* Fire belly badge pattern */}
        <path
          d="M 45 130 Q 65 110 75 130 Q 90 95 105 130 Q 115 110 135 130 L 135 152 C 135 166 45 166 45 152 Z"
          fill="#ff4b3e"
        />

        {/* Bottom rim */}
        <path d="M 45 150 C 45 164 135 164 135 150" fill="#d9480f" />

        {/* Fiery angry-cute determined eyebrows */}
        <path d="M 62 86 L 80 94" strokeWidth="4" />
        <path d="M 118 86 L 100 94" strokeWidth="4" />

        {/* Determined cartoon pie eyes */}
        <g fill="#1b1b2f">
          <ellipse cx="73" cy="104" rx="7" ry="9" stroke="none" />
          <polygon points="73,104 67,100 68,107" fill="#ffffff" stroke="none" />

          <ellipse cx="107" cy="104" rx="7" ry="9" stroke="none" />
          <polygon points="107,104 113,100 112,107" fill="#ffffff" stroke="none" />
        </g>

        {/* Warm blush */}
        <circle cx="60" cy="116" r="6" fill="#ff4b3e" opacity="0.9" stroke="none" />
        <circle cx="120" cy="116" r="6" fill="#ff4b3e" opacity="0.9" stroke="none" />

        {/* Fierce grinning smile showing teeth line */}
        <path
          d="M 74 122 Q 90 138 106 122 Z"
          fill="#ffffff"
        />
        <line x1="74" y1="122" x2="106" y2="122" strokeWidth="3" />
        <path d="M 76 122 Q 90 136 104 122" fill="none" strokeWidth="3" />
        <line x1="84" y1="122" x2="84" y2="128" strokeWidth="2" />
        <line x1="90" y1="122" x2="90" y2="130" strokeWidth="2" />
        <line x1="96" y1="122" x2="96" y2="128" strokeWidth="2" />
      </g>
    </svg>
  );
}

/**
 * Mascot registry map for mascot_svg_key from database / API
 */
export const MASCOT_REGISTRY: Record<string, React.ComponentType<MascotProps>> = {
  "can-soda-lemon": CanSodaLemon,
  "can-kopi-susu": CanKopiSusu,
  "can-teh-melati": CanTehMelati,
  "can-lite-zero": CanLiteZero,
  "can-buah-mangga": CanBuahMangga,
  "can-soda-jahe": CanSodaJahe,
};

export interface DynamicMascotProps extends MascotProps {
  name: string;
}

/**
 * MascotCan dispatcher: Renders the appropriate mascot SVG based on key.
 * Falls back to CanSodaLemon if key is not found.
 */
export function MascotCan({ name, ...props }: DynamicMascotProps) {
  const Component = MASCOT_REGISTRY[name] || CanSodaLemon;
  return <Component {...props} />;
}
