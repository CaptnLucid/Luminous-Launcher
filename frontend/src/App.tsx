// frontend/src/App.tsx
import React, { useEffect, useState } from "react";
import {
  CheckLauncherUpdates,
  LoadAvailableProfiles,
  ExecuteGame,
  ApplyApplicationUpdate, // 💡 Import our brand new update action module
} from "../wailsjs/go/backend/App";
import "./style.css";

// Interface for mapping response contracts cleanly
interface UpdateData {
  has_update: boolean;
  version: string;
  changelog: string;
  url: string;
  download_url: string;
}

export default function App() {
  const [profiles, setProfiles] = useState<Record<string, string>>({});
  const [selectedProfile, setSelectedProfile] = useState("");
  const [launchMode, setLaunchMode] = useState("pearl");
  const [customPath, setCustomPath] = useState("");
  const [affinity, setAffinity] = useState("FFFF");
  const [consoleLogs, setConsoleLogs] = useState<string[]>([]);
  const [isLaunching, setIsLaunching] = useState(false);

  // 💡 State hooks to track available system package revisions
  const [remoteUpdate, setRemoteUpdate] = useState<UpdateData | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    LoadAvailableProfiles().then((res) => {
      setProfiles(res || {});
      addLog("info", "System profile paths scanned.");
    });

    CheckLauncherUpdates()
      .then((updateInfo) => {
        if (updateInfo && updateInfo.has_update) {
          setRemoteUpdate(updateInfo); // Store update payload context data
          addLog("warn", `Update available — ${updateInfo.version}`);
        } else {
          addLog("info", "Launcher is up to date.");
        }
      })
      .catch((err) => addLog("error", `Update check failed: ${err}`));
  }, []);

  const addLog = (type: "info" | "warn" | "error" | "ok", message: string) => {
    const timestamp = new Date().toLocaleTimeString("en-US", { hour12: false });
    const entryString = `${type}|${timestamp}|${message}`;

    setConsoleLogs((prev) => {
      // If the log array has entries, check if the last message matches this one
      if (prev.length > 0) {
        const lastEntry = prev[prev.length - 1];
        const [, , lastMsg] = lastEntry.split("|");
        if (lastMsg === message) {
          return prev; // Duplicate blocked! Return unchanged state array
        }
      }
      return [...prev, entryString];
    });
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

  // 💡 Function to trigger the automated software rewrite sequence
  const handleSystemUpdate = async () => {
    if (!remoteUpdate) return;
    setIsUpdating(true);
    addLog("info", "Downloading new executable engine layer from GitHub...");

    // 💡 Clean backend mapping injection! No more .replace() modifications
    const statusResult = await ApplyApplicationUpdate(
      remoteUpdate.download_url,
    );

    if (statusResult !== "Success") {
      addLog("error", `Update sequence aborted: ${statusResult}`);
      setIsUpdating(false);
    }
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
            <span className="header-eyebrow">Custom BDO Launcher</span>
            <h1 className="header-title">
              LUMINOUS <span>LAUNCHER</span>
            </h1>
          </div>

          {/* Version badge section with our inline updater button trigger */}
          <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>
            {remoteUpdate && (
              <button
                onClick={handleSystemUpdate}
                disabled={isUpdating}
                className={`update-badge-btn ${isUpdating ? "processing" : ""}`}
              >
                {isUpdating
                  ? "UPDATING..."
                  : `INSTALL UPDATE ${remoteUpdate.version}`}
              </button>
            )}
            <span className="version-badge">v1.0.0</span>
          </div>
        </header>

        {/* Main */}
        <main className="main">
          {/* Left: Controls */}
          <div className="panel panel-left">
            <p className="section-label">Launch Configuration</p>

            <div className="field-group">
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

              <div className="field">
                <label className="field-label">
                  Nvidia Inspector Profile (.nip)
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
