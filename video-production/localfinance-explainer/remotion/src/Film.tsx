import React from "react";
import {
  AbsoluteFill,
  Easing,
  Img,
  Sequence,
  interpolate,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Audio } from "@remotion/media";
import { loadFont } from "@remotion/fonts";
import type { Caption } from "@remotion/captions";
import captionData from "./captions.json";

loadFont({
  family: "Inter",
  url: staticFile("inter-variable.woff2"),
  weight: "100 900",
});
const INK = "#07110d",
  MINT = "#69e7ad",
  PAPER = "#f4f7f3",
  MUTED = "#a9b9af",
  GREEN = "#11754b";
const clamp = {
  extrapolateLeft: "clamp" as const,
  extrapolateRight: "clamp" as const,
};
const ease = { ...clamp, easing: Easing.bezier(0.16, 1, 0.3, 1) };
const box: React.CSSProperties = {
  background: "#10241a",
  border: "2px solid #284034",
  borderRadius: 24,
  padding: 38,
};
const label: React.CSSProperties = {
  fontSize: 30,
  letterSpacing: 4,
  fontWeight: 600,
  textTransform: "uppercase",
};

function Icon({
  type,
  size = 64,
  color = "currentColor",
}: {
  type: string;
  size?: number;
  color?: string;
}) {
  const paths: Record<string, React.ReactNode> = {
    file: (
      <>
        <path d="M7 3h7l4 4v14H7z" />
        <path d="M14 3v5h4M10 12h5m-5 4h5" />
      </>
    ),
    check: <path d="m5 12 5 5L20 6" />,
    arrow: <path d="M4 12h16m-6-6 6 6-6 6" />,
    lock: (
      <>
        <rect x="5" y="10" width="14" height="11" rx="2" />
        <path d="M8 10V7a4 4 0 0 1 8 0v3m-4 5v2" />
      </>
    ),
    laptop: (
      <>
        <rect x="3" y="3" width="18" height="14" rx="2" />
        <path d="M1 21h22M8 17v4m8-4v4" />
      </>
    ),
    tag: (
      <>
        <path d="m3 3 9 0 10 10-9 9L3 12z" />
        <circle cx="8" cy="8" r="1" />
      </>
    ),
    db: (
      <>
        <ellipse cx="12" cy="5" rx="8" ry="3" />
        <path d="M4 5v14c0 4 16 4 16 0V5M4 12c0 4 16 4 16 0" />
      </>
    ),
  };
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {paths[type] || paths.check}
    </svg>
  );
}

function Reveal({
  children,
  delay = 0,
  style = {},
}: {
  children: React.ReactNode;
  delay?: number;
  style?: React.CSSProperties;
}) {
  const f = useCurrentFrame();
  return (
    <div
      style={{
        ...style,
        opacity: interpolate(f, [delay, delay + 20], [0, 1], clamp),
        translate: `0 ${interpolate(f, [delay, delay + 30], [44, 0], ease)}px`,
      }}
    >
      {children}
    </div>
  );
}
function Header({ step, dark = true }: { step: string; dark?: boolean }) {
  return (
    <div
      style={{
        position: "absolute",
        top: 125,
        left: 88,
        right: 88,
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        color: dark ? MINT : GREEN,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: 15,
          alignItems: "center",
          fontSize: 29,
          fontWeight: 600,
        }}
      >
        <Img src={staticFile("mark.svg")} style={{ width: 44, height: 44 }} />
        LocalFinance
      </div>
      <span style={{ fontSize: 25, letterSpacing: 3 }}>{step}</span>
    </div>
  );
}
function Title({
  top,
  lines,
  dark = true,
}: {
  top?: string;
  lines: string[];
  dark?: boolean;
}) {
  return (
    <Reveal style={{ position: "absolute", left: 88, top: 285, right: 72 }}>
      {top && (
        <div style={{ ...label, color: dark ? MINT : GREEN, marginBottom: 26 }}>
          {top}
        </div>
      )}
      <h1
        style={{
          margin: 0,
          fontSize: 100,
          fontWeight: 600,
          lineHeight: 1.03,
          letterSpacing: -5,
          color: dark ? PAPER : INK,
        }}
      >
        {lines.map((s, i) => (
          <React.Fragment key={s}>
            {i > 0 && <br />}
            <span
              style={{
                color:
                  i === lines.length - 1 ? (dark ? MINT : GREEN) : undefined,
              }}
            >
              {s}
            </span>
          </React.Fragment>
        ))}
      </h1>
    </Reveal>
  );
}
function Foot({
  dark = true,
  demo = true,
}: {
  dark?: boolean;
  demo?: boolean;
}) {
  return (
    <div
      style={{
        position: "absolute",
        left: 88,
        right: 88,
        top: 1568,
        fontSize: 24,
        letterSpacing: 1.5,
        color: dark ? "#93a69a" : "#536a5b",
        display: "flex",
        justifyContent: "space-between",
      }}
    >
      <span>
        {demo ? "ILLUSTRATIVE DEMO · FICTIONAL DATA" : "PRIVATE BY DESIGN"}
      </span>
      <span>LOCALFINANCE</span>
    </div>
  );
}
function Captions() {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const caption = (captionData as Caption[]).find(
    (c) => (frame / fps) * 1000 >= c.startMs && (frame / fps) * 1000 < c.endMs,
  );
  return caption ? (
    <div
      style={{
        position: "absolute",
        left: 88,
        right: 88,
        top: 1670,
        minHeight: 120,
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        textAlign: "center",
      }}
    >
      <span
        style={{
          fontSize: 39,
          fontWeight: 500,
          lineHeight: 1.34,
          color: "#fff",
          background: "#07110deb",
          borderRadius: 14,
          padding: "16px 24px",
          maxWidth: "100%",
        }}
      >
        {caption.text}
      </span>
    </div>
  ) : null;
}

function Statement({
  title,
  format,
  index,
}: {
  title: string;
  format: string;
  index: number;
}) {
  const f = useCurrentFrame();
  return (
    <div
      style={{
        position: "absolute",
        width: 680,
        height: 345,
        padding: 40,
        borderRadius: 14,
        background: PAPER,
        color: INK,
        left: 110 + index * 35,
        top: 800 + index * 172,
        boxShadow: "0 26px 60px #0005",
        rotate: `${interpolate(f, [0, 105], [[-9, 7, -4][index], 0], ease)}deg`,
        translate: `${interpolate(f, [0, 90], [[-60, 80, -35][index], 0], ease)}px 0`,
        opacity: interpolate(f, [0 + index * 6, 18 + index * 6], [0, 1], clamp),
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          borderBottom: "2px solid #c9d7ce",
          paddingBottom: 24,
        }}
      >
        <span style={{ fontSize: 35, fontWeight: 600 }}>{title}</span>
        <span style={{ fontSize: 24, color: GREEN }}>{format}</span>
      </div>
      {[0, 1, 2].map((x) => (
        <div
          key={x}
          style={{
            display: "flex",
            justifyContent: "space-between",
            marginTop: 22,
          }}
        >
          <span
            style={{ width: 230 + x * 35, height: 12, background: "#d2ddd5" }}
          />
          <span
            style={{
              width: 94,
              height: 12,
              background: x === 2 ? "#16b978" : "#a9bdb0",
            }}
          />
        </div>
      ))}
    </div>
  );
}
function Opening() {
  return (
    <AbsoluteFill style={{ background: INK }}>
      <Header step="THE BIG PICTURE" />
      <Title lines={["Your money", "tells a story."]} />
      <Reveal
        delay={18}
        style={{
          position: "absolute",
          top: 570,
          left: 88,
          right: 120,
          fontSize: 43,
          lineHeight: 1.4,
          color: MUTED,
        }}
      >
        Find the meaning
        <br />
        between the transactions.
      </Reveal>
      <Statement title="Bank statement" format="PDF" index={0} />
      <Statement title="Credit card bill" format="CSV" index={1} />
      <Statement title="Monthly transactions" format="EXCEL" index={2} />
      <Foot demo={false} />
    </AbsoluteFill>
  );
}
function Intro() {
  const f = useCurrentFrame();
  return (
    <AbsoluteFill style={{ background: PAPER }}>
      <Header step="MEET YOUR WORKSPACE" dark={false} />
      <Title lines={["Meet", "LocalFinance."]} dark={false} />
      <Reveal
        delay={18}
        style={{
          position: "absolute",
          top: 570,
          left: 88,
          fontSize: 44,
          lineHeight: 1.4,
          color: "#536a5b",
        }}
      >
        Your private money workspace.
      </Reveal>
      <Reveal
        delay={8}
        style={{
          position: "absolute",
          left: 88,
          top: 800,
          width: 904,
          height: 570,
          borderRadius: 30,
          border: "3px solid #b8cdbf",
          background: INK,
          boxShadow: "0 40px 70px #07110d20",
          scale: interpolate(f, [0, 200], [0.96, 1.02], ease),
        }}
      >
        <div
          style={{
            position: "absolute",
            inset: "40px 44px",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <Img
            src={staticFile("mark.svg")}
            style={{ width: 190, height: 190 }}
          />
          <div
            style={{
              fontSize: 50,
              color: PAPER,
              fontWeight: 600,
              marginTop: 26,
            }}
          >
            One place for your records.
          </div>
          <div style={{ fontSize: 34, color: MINT, marginTop: 24 }}>
            Bank accounts + Credit cards
          </div>
        </div>
      </Reveal>
      <div
        style={{
          position: "absolute",
          left: 340,
          top: 1370,
          width: 400,
          height: 18,
          borderRadius: "0 0 24px 24px",
          background: "#b8cdbf",
        }}
      />
      <Reveal
        delay={65}
        style={{
          position: "absolute",
          top: 1460,
          left: 88,
          right: 88,
          textAlign: "center",
          fontSize: 34,
          color: GREEN,
        }}
      >
        Offline. On your computer.
      </Reveal>
      <Foot dark={false} demo={false} />
    </AbsoluteFill>
  );
}
function ImportScene() {
  const f = useCurrentFrame();
  const imported = f >= 155;
  return (
    <AbsoluteFill style={{ background: INK }}>
      <Header step="01 / IMPORT" />
      <Title lines={["Statements in.", "Clarity out."]} />
      <Reveal
        delay={12}
        style={{
          position: "absolute",
          top: 570,
          left: 88,
          fontSize: 40,
          color: MUTED,
        }}
      >
        Supported PDF, CSV & Excel files.
      </Reveal>
      <Reveal
        delay={15}
        style={{ position: "absolute", top: 740, left: 88, width: 904 }}
      >
        <div
          style={{
            display: "flex",
            gap: 28,
            alignItems: "center",
            padding: "30px 36px",
            background: "#183b28",
            borderRadius: 20,
            border: "2px solid #326348",
          }}
        >
          <Icon type="file" color={MINT} />
          <div>
            <div style={{ fontSize: 35, fontWeight: 600 }}>
              monthly-statement.pdf
            </div>
            <div style={{ fontSize: 27, color: MUTED, marginTop: 9 }}>
              Example supported statement
            </div>
          </div>
        </div>
        <div
          style={{
            display: "flex",
            justifyContent: "center",
            height: 105,
            alignItems: "center",
            color: MINT,
            rotate: "90deg",
          }}
        >
          <Icon type="arrow" size={50} />
        </div>
        <div style={{ ...box, padding: 38 }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: 35,
            }}
          >
            <strong style={{ fontSize: 40 }}>
              {imported ? "Imported to your ledger" : "Preview transactions"}
            </strong>
            <span style={{ fontSize: 26, color: MINT }}>
              {imported ? "✓ Saved" : "REVIEW"}
            </span>
          </div>
          {[
            ["Salary", "+ ₹75,000"],
            ["Groceries", "− ₹4,800"],
            ["Dining", "− ₹1,200"],
          ].map(([s, v], i) => (
            <Reveal
              key={s}
              delay={40 + i * 17}
              style={{
                display: "flex",
                justifyContent: "space-between",
                padding: "25px 0",
                borderTop: "1px solid #304237",
                fontSize: 36,
              }}
            >
              <span>{s}</span>
              <strong
                style={{ fontWeight: 500, color: i === 0 ? MINT : PAPER }}
              >
                {v}
              </strong>
            </Reveal>
          ))}
          <Reveal
            delay={100}
            style={{
              marginTop: 28,
              padding: 25,
              background: imported ? MINT : "#1b4b32",
              color: imported ? INK : MINT,
              borderRadius: 12,
              display: "flex",
              justifyContent: "center",
              gap: 18,
              fontSize: 32,
              fontWeight: 600,
            }}
          >
            <Icon type={imported ? "check" : "arrow"} size={38} />
            {imported ? "Records saved locally" : "Import reviewed records"}
          </Reveal>
        </div>
      </Reveal>
      <Foot />
    </AbsoluteFill>
  );
}
function Organize() {
  const f = useCurrentFrame();
  const sorted = f > 65;
  return (
    <AbsoluteFill style={{ background: PAPER }}>
      <Header step="02 / ORGANIZE" dark={false} />
      <Title lines={["A ledger that", "stays yours."]} dark={false} />
      <Reveal
        delay={10}
        style={{
          position: "absolute",
          top: 570,
          left: 88,
          right: 88,
          fontSize: 42,
          color: "#536a5b",
        }}
      >
        Rules help. You stay in control.
      </Reveal>
      <Reveal
        delay={10}
        style={{
          position: "absolute",
          top: 770,
          left: 88,
          right: 88,
          background: "#fff",
          border: "2px solid #cfdbd2",
          borderRadius: 26,
          padding: 42,
          boxShadow: "0 30px 70px #07110d0c",
          color: INK,
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            fontSize: 44,
            fontWeight: 600,
          }}
        >
          <span>Corner Café</span>
          <span>₹1,200</span>
        </div>
        <div style={{ marginTop: 24, fontSize: 29, color: "#64786a" }}>
          A transaction you can review and refine.
        </div>
        <div style={{ height: 1, background: "#d6e1d9", margin: "35px 0" }} />
        <div style={{ fontSize: 27, color: "#64786a", marginBottom: 17 }}>
          CATEGORY
        </div>
        <div
          style={{
            fontSize: 42,
            fontWeight: 600,
            color: sorted ? GREEN : "#64786a",
            display: "flex",
            gap: 18,
            alignItems: "center",
          }}
        >
          <Icon type="tag" size={40} />
          {sorted ? "Food & Dining" : "Uncategorized"}
        </div>
        <Reveal
          delay={100}
          style={{
            marginTop: 32,
            display: "flex",
            gap: 16,
            fontSize: 28,
            color: GREEN,
          }}
        >
          <span
            style={{
              padding: "14px 22px",
              borderRadius: 8,
              background: "#e0f4e7",
            }}
          >
            weekend
          </span>
          <span
            style={{
              padding: "14px 22px",
              borderRadius: 8,
              background: "#e0f4e7",
            }}
          >
            personal
          </span>
        </Reveal>
        <Reveal
          delay={130}
          style={{
            fontSize: 32,
            marginTop: 30,
            paddingTop: 30,
            borderTop: "1px solid #d6e1d9",
            color: "#536a5b",
          }}
        >
          Note: Sunday lunch with friends
        </Reveal>
      </Reveal>
      <Reveal
        delay={185}
        style={{
          position: "absolute",
          left: 88,
          right: 88,
          top: 1350,
          padding: 32,
          borderRadius: 16,
          background: "#dff8e9",
          color: GREEN,
          display: "flex",
          gap: 20,
          alignItems: "center",
          fontSize: 34,
          fontWeight: 600,
        }}
      >
        <Icon type="check" size={46} />
        Reimported. Your edits stay.
      </Reveal>
      <Reveal
        delay={195}
        style={{
          position: "absolute",
          top: 1490,
          left: 88,
          fontSize: 31,
          color: "#536a5b",
        }}
      >
        Same statement. No duplicate transactions.
      </Reveal>
      <Foot dark={false} />
    </AbsoluteFill>
  );
}
function Understand() {
  const f = useCurrentFrame();
  return (
    <AbsoluteFill style={{ background: INK }}>
      <Header step="03 / UNDERSTAND" />
      <Title lines={["From transactions", "to perspective."]} />
      <Reveal
        delay={8}
        style={{
          position: "absolute",
          top: 580,
          left: 88,
          right: 88,
          fontSize: 40,
          color: MUTED,
        }}
      >
        Spending. Cash flow. Budgets.
      </Reveal>
      <Reveal
        delay={12}
        style={{ position: "absolute", top: 730, left: 88, right: 88 }}
      >
        <div
          style={{
            display: "flex",
            gap: 44,
            borderBottom: "1px solid #304237",
            paddingBottom: 36,
          }}
        >
          {[
            ["INCOME", "₹75,000"],
            ["EXPENSES", "₹42,000"],
          ].map(([k, v]) => (
            <div key={k} style={{ flex: 1 }}>
              <div style={{ ...label, fontSize: 25, color: MUTED }}>{k}</div>
              <div
                style={{
                  fontSize: 55,
                  fontWeight: 600,
                  letterSpacing: -2,
                  marginTop: 20,
                  color: k === "INCOME" ? MINT : PAPER,
                }}
              >
                {v}
              </div>
            </div>
          ))}
        </div>
        <div style={{ fontSize: 34, margin: "35px 0 18px" }}>
          Spending by category
        </div>
        {[
          ["Home", 18000],
          ["Food", 12000],
          ["Travel", 7500],
          ["Other", 4500],
        ].map(([name, n], i) => (
          <div key={name} style={{ marginTop: 20 }}>
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                fontSize: 30,
                color: i === 0 ? PAPER : MUTED,
                marginBottom: 13,
              }}
            >
              <span>{name}</span>
              <span>₹{Number(n).toLocaleString("en-IN")}</span>
            </div>
            <div
              style={{
                height: 20,
                background: "#1a3023",
                borderRadius: 5,
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  height: "100%",
                  width: `${interpolate(f, [40 + i * 15, 95 + i * 15], [0, (Number(n) / 18000) * 100], ease)}%`,
                  background:
                    i === 0 ? MINT : ["", "#9cbaa8", "#688c77", "#3d624d"][i],
                }}
              />
            </div>
          </div>
        ))}
        <Reveal
          delay={185}
          style={{ marginTop: 38, ...box, padding: "24px 32px" }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              fontSize: 29,
            }}
          >
            <strong>Food budget</strong>
            <span style={{ color: MINT }}>₹12,000 / ₹15,000</span>
          </div>
          <div
            style={{
              height: 13,
              background: "#284034",
              marginTop: 24,
              borderRadius: 6,
              overflow: "hidden",
            }}
          >
            <div
              style={{
                height: "100%",
                background: MINT,
                width: `${interpolate(f, [195, 240], [0, 80], ease)}%`,
              }}
            />
          </div>
          <div style={{ marginTop: 18, fontSize: 27, color: MUTED }}>
            80% of your chosen monthly limit
          </div>
        </Reveal>
      </Reveal>
      <Foot />
    </AbsoluteFill>
  );
}
function Local() {
  const f = useCurrentFrame();
  return (
    <AbsoluteFill style={{ background: PAPER }}>
      <Header step="ON YOUR MACHINE" dark={false} />
      <Title lines={["Built around", "your privacy."]} dark={false} />
      <Reveal
        delay={12}
        style={{
          position: "absolute",
          top: 570,
          left: 88,
          right: 88,
          fontSize: 42,
          color: "#536a5b",
        }}
      >
        Your records stay with you.
      </Reveal>
      <Reveal
        delay={14}
        style={{
          position: "absolute",
          left: 88,
          right: 88,
          top: 750,
          height: 680,
          border: "3px solid #a5bfae",
          borderRadius: 30,
          padding: "32px 40px",
          background: "#e7eee8",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 18,
            color: GREEN,
            fontSize: 32,
            marginBottom: 28,
          }}
        >
          <Icon type="laptop" size={44} />
          YOUR COMPUTER
        </div>
        {[
          ["laptop", "LocalFinance app", "Your workspace"],
          ["arrow", "Local Go engine", "Reads and organizes records"],
          ["db", "SQLite database", "Stores your ledger locally"],
        ].map(([type, title, sub], i) => (
          <React.Fragment key={title}>
            {i > 0 && (
              <div
                style={{
                  height: 36,
                  marginLeft: 40,
                  borderLeft: "2px solid #7fa48c",
                  opacity: interpolate(
                    f,
                    [30 + i * 30, 60 + i * 30],
                    [0, 1],
                    clamp,
                  ),
                }}
              />
            )}
            <Reveal
              delay={25 + i * 35}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 25,
                padding: "24px 28px",
                background: i === 2 ? INK : "#fbfdfb",
                borderRadius: 14,
                color: i === 2 ? PAPER : INK,
                border: "1px solid #c2d2c6",
              }}
            >
              <Icon type={type} color={i === 2 ? MINT : GREEN} size={46} />
              <div>
                <div style={{ fontSize: 36, fontWeight: 600 }}>{title}</div>
                <div
                  style={{
                    fontSize: 26,
                    marginTop: 9,
                    color: i === 2 ? MUTED : "#536a5b",
                  }}
                >
                  {sub}
                </div>
              </div>
            </Reveal>
          </React.Fragment>
        ))}
      </Reveal>
      <Reveal
        delay={160}
        style={{
          position: "absolute",
          left: 88,
          right: 88,
          top: 1470,
          fontSize: 30,
          color: GREEN,
        }}
      >
        No cloud sync · No bank connection
      </Reveal>
      <Foot dark={false} demo={false} />
      <Reveal
        delay={230}
        style={{
          position: "absolute",
          left: 88,
          right: 88,
          top: 1518,
          fontSize: 24,
          color: "#536a5b",
        }}
      >
        Public release checks are optional in Settings.
      </Reveal>
    </AbsoluteFill>
  );
}
function Close() {
  return (
    <AbsoluteFill style={{ background: MINT, color: INK }}>
      <Header step="YOUR NEXT CHAPTER" dark={false} />
      <Reveal style={{ position: "absolute", top: 410, left: 88 }}>
        <Img src={staticFile("mark.svg")} style={{ width: 150, height: 150 }} />
      </Reveal>
      <Reveal
        delay={10}
        style={{
          position: "absolute",
          top: 625,
          left: 88,
          fontSize: 64,
          fontWeight: 600,
          letterSpacing: -2,
        }}
      >
        LocalFinance
      </Reveal>
      <Reveal
        delay={20}
        style={{
          position: "absolute",
          top: 750,
          left: 88,
          right: 60,
          fontSize: 124,
          fontWeight: 600,
          lineHeight: 1.02,
          letterSpacing: -7,
        }}
      >
        Your money.
        <br />
        <span style={{ color: GREEN }}>Made clear.</span>
      </Reveal>
      <Reveal
        delay={45}
        style={{
          position: "absolute",
          top: 1110,
          left: 88,
          right: 88,
          fontSize: 41,
          padding: "35px 0",
          borderTop: "2px solid #07110d33",
          borderBottom: "2px solid #07110d33",
        }}
      >
        Import. &nbsp; Review. &nbsp; Understand.
      </Reveal>
      <Reveal
        delay={90}
        style={{
          position: "absolute",
          top: 1340,
          left: 88,
          right: 88,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          background: INK,
          color: PAPER,
          padding: "35px 40px",
          fontSize: 38,
          fontWeight: 600,
        }}
      >
        <span>Download LocalFinance</span>
        <Icon type="arrow" size={50} color={MINT} />
      </Reveal>
      <div
        style={{
          position: "absolute",
          left: 88,
          top: 1515,
          fontSize: 29,
          color: GREEN,
        }}
      >
        Windows · macOS · Linux
      </div>
      <Foot dark={false} demo={false} />
    </AbsoluteFill>
  );
}
function Progress() {
  const f = useCurrentFrame();
  return (
    <div
      style={{
        position: "absolute",
        bottom: 0,
        left: 0,
        width: `${(f / 1799) * 100}%`,
        height: 8,
        background: GREEN,
      }}
    />
  );
}
export function Film() {
  return (
    <AbsoluteFill
      style={{
        fontFamily: "Inter, sans-serif",
        color: PAPER,
        fontVariantNumeric: "tabular-nums",
      }}
    >
      <Sequence from={0} durationInFrames={180} name="The question">
        <Opening />
      </Sequence>
      <Sequence from={180} durationInFrames={210} name="Meet LocalFinance">
        <Intro />
      </Sequence>
      <Sequence from={390} durationInFrames={300} name="Import statements">
        <ImportScene />
      </Sequence>
      <Sequence
        from={690}
        durationInFrames={270}
        name="Organize and keep edits"
      >
        <Organize />
      </Sequence>
      <Sequence from={960} durationInFrames={330} name="Understand spending">
        <Understand />
      </Sequence>
      <Sequence from={1290} durationInFrames={300} name="Local architecture">
        <Local />
      </Sequence>
      <Sequence from={1590} durationInFrames={210} name="Brand and download">
        <Close />
      </Sequence>
      <Audio src={staticFile("mix.flac")} />
      <Captions />
      <Progress />
    </AbsoluteFill>
  );
}
