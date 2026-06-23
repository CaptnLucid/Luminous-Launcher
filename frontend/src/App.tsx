// frontend/src/App.tsx
import React, { useEffect, useState } from "react";
import {
  CheckLauncherUpdates,
  LoadAvailableProfiles,
  ExecuteGame,
} from "../wailsjs/go/backend/App";
import "./style.css";

export default function App() {
  const [profiles, setProfiles] = useState<Record<string, string>>({});
  const [selectedProfile, setSelectedProfile] = useState("");
  const [launchMode, setLaunchMode] = useState("pearl");
  const [customPath, setCustomPath] = useState("");
  const [affinity, setAffinity] = useState("FFFF");
  const [consoleLogs, setConsoleLogs] = useState<string[]>([]);
  const [isLaunching, setIsLaunching] = useState(false);

  useEffect(() => {
    LoadAvailableProfiles().then((res) => {
      setProfiles(res || {});
      addLog("info", "System profile paths scanned.");
    });

    CheckLauncherUpdates()
      .then((updateInfo) => {
        if (updateInfo && updateInfo.has_update) {
          addLog("warn", `Update available — v${updateInfo.version}`);
          alert(
            `New Update Available: ${updateInfo.version}\n\nChangelog:\n${updateInfo.changelog}\n\nDownload at: ${updateInfo.url}`,
          );
        } else {
          addLog("info", "Launcher is up to date.");
        }
      })
      .catch((err) => addLog("error", `Update check failed: ${err}`));
  }, []);

  const addLog = (type: "info" | "warn" | "error" | "ok", message: string) => {
    const timestamp = new Date().toLocaleTimeString("en-US", { hour12: false });
    setConsoleLogs((prev) => [...prev, `${type}|${timestamp}|${message}`]);
  };

  const handleLaunch = async () => {
    setIsLaunching(true);
    addLog("info", "Initializing launch sequence...");
    const activeNipPath = profiles[selectedProfile] || "";
    if (activeNipPath) addLog("info", `Profile loaded: ${selectedProfile}.nip`);

    const result = await ExecuteGame(
      launchMode,
      customPath,
      activeNipPath,
      affinity,
    );
    addLog("ok", `${result}`);
    setIsLaunching(false);
  };

  const logColor: Record<string, string> = {
    info: "#6b7280",
    warn: "#f59e0b",
    error: "#ef4444",
    ok: "#10b981",
  };

  return (
    <>
      <div className="app">
        {/* Header */}
        <header className="header">
          <div className="header-left">
            <span className="header-eyebrow">BDO Launcher</span>
            <h1 className="header-title">
              LUMINOUS <span>LAUNCHER</span>
            </h1>
          </div>
          <span className="version-badge">v1.0.0</span>
        </header>

        {/* Main */}
        <main className="main">
          {/* Left: Controls */}
          <div className="panel panel-left">
            <p className="section-label">Launch Configuration</p>

            <div className="field-group">
              {/* Launch mode + Affinity side by side */}
              <div className="field-row">
                <div className="field">
                  <label className="field-label">Distribution Target</label>
                  <select
                    value={launchMode}
                    onChange={(e) => setLaunchMode(e.target.value)}
                  >
                    <option value="pearl">Pearl Abyss Launcher</option>
                    <option value="steam">Steam</option>
                    <option value="custom">Custom Path</option>
                  </select>
                </div>

                <div className="field">
                  <label className="field-label">CPU Affinity Mask</label>
                  <input
                    type="text"
                    className="mono"
                    value={affinity}
                    onChange={(e) => setAffinity(e.target.value)}
                    placeholder="FFFF"
                  />
                </div>
              </div>

              {/* Custom path — conditional */}
              {launchMode === "custom" && (
                <div className="field field-slide">
                  <label className="field-label">Executable Path</label>
                  <input
                    type="text"
                    value={customPath}
                    onChange={(e) => setCustomPath(e.target.value)}
                    placeholder="D:\BlackDesert\BlackDesertLauncher.exe"
                  />
                </div>
              )}

              <div className="divider" />

              {/* NIP Profile */}
              <div className="field">
                <label className="field-label">
                  Nvidia Inspector Profile{" "}
                  <span style={{ color: "#71717a", fontWeight: 400 }}>
                    (.nip)
                  </span>
                </label>
                <select
                  value={selectedProfile}
                  onChange={(e) => setSelectedProfile(e.target.value)}
                >
                  <option value="">No profile — skip injection</option>
                  {Object.keys(profiles).map((name) => (
                    <option key={name} value={name}>
                      {name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* Right: Log + Launch */}
          <div className="panel-right">
            {/* Log */}
            <div className="log-area">
              <p className="section-label">Runtime Log</p>
              <div className="log-scroll">
                {consoleLogs.length === 0 && (
                  <span
                    style={{
                      fontSize: 11,
                      color: "#3f3f46",
                      fontFamily: "'JetBrains Mono', monospace",
                    }}
                  >
                    Awaiting activity...
                  </span>
                )}
                {consoleLogs.map((entry, i) => {
                  const [type, time, ...rest] = entry.split("|");
                  return (
                    <div className="log-entry" key={i}>
                      <span className="log-time">{time}</span>
                      <span
                        className="log-dot"
                        style={{ background: logColor[type] || "#6b7280" }}
                      />
                      <span className="log-msg">{rest.join("|")}</span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Launch */}
            <div className="launch-area">
              <button
                className={`launch-btn${isLaunching ? " launching" : ""}`}
                onClick={handleLaunch}
                disabled={isLaunching}
              >
                {isLaunching ? "LAUNCHING..." : "LAUNCH GAME"}
              </button>
            </div>
          </div>
        </main>

        {/* Footer */}
        <footer className="footer">
          <span className="footer-note">
            Place <code>/nvidiaProfileInspector</code> and{" "}
            <code>/profiles</code> adjacent to this executable
          </span>
          <div className="status-dot" title="System ready" />
        </footer>
      </div>
    </>
  );
}
