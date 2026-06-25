// frontend/src/App.tsx
import React, { useEffect, useState } from "react";
import {
  CheckLauncherUpdates,
  LoadAvailableProfiles,
  ExecuteGame,
  ApplyApplicationUpdate,
  ApplyNipProfile,
  GetCurrentVersion,
  SaveSettingUpdate,
  GetSettings,
  DetectCPUMask,
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
  const [isApplyingProfile, setIsApplyingProfile] = useState(false);
  const [currentVersion, setCurrentVersion] = useState("1.0.0");
  const [executablePath, setExecutablePath] = useState("");
  const [isSteam, setIsSteam] = useState(false);
  const [affinityMask, setAffinityMask] = useState("0");

  // State hooks to track available system package revisions
  const [remoteUpdate, setRemoteUpdate] = useState<UpdateData | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    LoadAvailableProfiles().then((res) => {
      setProfiles(res || {});
      addLog("info", "System profile paths scanned.");
    });

    GetCurrentVersion()
      .then((v: string) => {
        setCurrentVersion(v);
      })
      .catch((err) => {
        console.error(err);
      });

    // Fetch and seed local persistent user configurations on boot
    GetSettings()
      .then((config) => {
        if (config) {
          setExecutablePath(config.executablePath);
          setCustomPath(config.executablePath);
          setIsSteam(config.isSteam);

          const savedMask = String(config.affinityMask || "0");
          setAffinityMask(savedMask);

          if (savedMask !== "0" && savedMask !== "") {
            setAffinity(savedMask.toUpperCase());
          }

          if (config.selectedProfile) {
            setSelectedProfile(config.selectedProfile);
          }

          if (config.executablePath) {
            setLaunchMode("custom");
          } else if (config.isSteam) {
            setLaunchMode("steam");
          }
        }
      })
      .catch((err) => console.error("Failed to load settings:", err));

    CheckLauncherUpdates()
      .then((updateInfo) => {
        if (updateInfo && updateInfo.has_update) {
          setRemoteUpdate(updateInfo);
          addLog("warn", `Update available — ${updateInfo.version}`);
        } else {
          addLog("info", "Launcher is up to date.");
        }
      })
      .catch((err) => addLog("error", `Update check failed: ${err}`));
  }, []);

  const handleSettingsChange = (
    path: string,
    steamToggle: boolean,
    maskStr: string,
    profile: string,
  ) => {
    setExecutablePath(path);
    setCustomPath(path);
    setIsSteam(steamToggle);
    setAffinityMask(maskStr);
    SaveSettingUpdate(path, steamToggle, maskStr, profile);
  };

  const addLog = (type: "info" | "warn" | "error" | "ok", message: string) => {
    const timestamp = new Date().toLocaleTimeString("en-US", { hour12: false });
    const entryString = `${type}|${timestamp}|${message}`;

    setConsoleLogs((prev) => {
      if (prev.length > 0) {
        const lastEntry = prev[prev.length - 1];
        const [, , lastMsg] = lastEntry.split("|");
        if (lastMsg === message) {
          return prev;
        }
      }
      return [...prev, entryString];
    });
  };

  const handleLaunch = async () => {
    setIsLaunching(true);
    addLog("info", "Initializing launch sequence...");

    const finalMode = launchMode === "custom" && isSteam ? "steam" : launchMode;
    const finalPath = launchMode === "custom" ? executablePath : customPath;

    const finalAffinity =
      affinityMask !== "0" && affinityMask !== "" ? affinityMask : affinity;

    const result = await ExecuteGame(
      finalMode,
      finalPath,
      finalAffinity.toUpperCase(),
    );
    addLog("ok", `${result}`);
    setIsLaunching(false);
  };

  const handleApplyProfile = async () => {
    if (!selectedProfile) {
      addLog("warn", "No profile selected.");
      return;
    }
    const nipPath = profiles[selectedProfile] || "";
    setIsApplyingProfile(true);
    addLog("info", `Injecting profile: ${selectedProfile}.nip`);
    const result = await ApplyNipProfile(nipPath);
    if (result === "Success") {
      addLog("ok", `Profile applied: ${selectedProfile}`);
    } else {
      addLog("error", result);
    }
    setIsApplyingProfile(false);
  };

  const handleSystemUpdate = async () => {
    if (!remoteUpdate) return;
    setIsUpdating(true);
    addLog("info", "Downloading new executable engine layer from GitHub...");

    const statusResult = await ApplyApplicationUpdate(
      remoteUpdate.download_url,
    );

    if (statusResult !== "Success") {
      addLog("error", `Update sequence aborted: ${statusResult}`);
      setIsUpdating(false);
    }
  };

  const handleAutoDetectCPU = async () => {
    addLog(
      "info",
      "Analyzing system hardware topology for optimal core assignment...",
    );
    try {
      // Cast to 'any' to bypass the stale autogenerated TypeScript string binding
      const result = (await DetectCPUMask()) as any;

      // Dump every internal Go step straight into your visible Runtime Log UI element
      if (result && Array.isArray(result.logs)) {
        result.logs.forEach((goLogLine: string) => {
          addLog("info", `[Go Engine] ${goLogLine}`);
        });
      }

      const optimalHex = result.mask;
      setAffinity(optimalHex);
      addLog(
        "ok",
        `Optimal CPU Affinity Profile applied to field: 0x${optimalHex}`,
      );

      if (affinityMask !== "0" && affinityMask !== "") {
        handleSettingsChange(
          executablePath,
          isSteam,
          optimalHex,
          selectedProfile,
        );
      }
    } catch (err) {
      addLog(
        "error",
        "Failed communicating with native hardware processor arrangement hooks.",
      );
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
            <span className="version-badge">{currentVersion}</span>
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
                    <option value="pearl">
                      Pearl Abyss Launcher (Standalone)
                    </option>
                    <option value="steam">Steam Client</option>
                    <option value="custom">Custom Path</option>
                  </select>
                </div>
                {/* CPU Affinity Field */}
                <div className="field">
                  <label className="field-label">CPU Affinity Mask</label>
                  <div className="affinity-input-wrapper">
                    <input
                      type="text"
                      className="mono"
                      value={affinity}
                      disabled={affinityMask !== "0" && affinityMask !== ""}
                      onChange={(e) => {
                        const uppercaseVal = e.target.value.toUpperCase();
                        setAffinity(uppercaseVal);
                        if (affinityMask !== "0" && affinityMask !== "") {
                          handleSettingsChange(
                            executablePath,
                            isSteam,
                            uppercaseVal,
                            selectedProfile,
                          );
                        }
                      }}
                      placeholder="FFFF"
                    />

                    {/* Auto-Detect Activation Action Trigger Badge */}
                    <button
                      className="cpu-detect-inline-btn"
                      onClick={handleAutoDetectCPU}
                      title="Auto Detect Optimal Game Cores"
                    >
                      AUTO
                    </button>

                    <label
                      className="checkbox-container affinity-lock-check"
                      title="Lock Affinity Profile"
                    >
                      <input
                        type="checkbox"
                        checked={affinityMask !== "0" && affinityMask !== ""}
                        onChange={(e) => {
                          const nextMask = e.target.checked ? affinity : "0";
                          handleSettingsChange(
                            executablePath,
                            isSteam,
                            nextMask,
                            selectedProfile,
                          );
                        }}
                      />
                      <span className="checkmark"></span>
                    </label>
                  </div>
                </div>
              </div>

              {launchMode === "custom" && (
                <div className="field field-slide">
                  <label className="field-label">
                    Executable Path Override
                  </label>
                  <input
                    type="text"
                    value={executablePath}
                    onChange={(e) =>
                      handleSettingsChange(
                        e.target.value,
                        isSteam,
                        affinityMask,
                        selectedProfile,
                      )
                    }
                    placeholder="D:\BlackDesert\BlackDesertLauncher.exe"
                  />

                  {/* Platform Mode Checkbox selectors */}
                  <div className="platform-toggle-row">
                    <label className="checkbox-container">
                      <input
                        type="checkbox"
                        checked={!isSteam}
                        onChange={() =>
                          handleSettingsChange(
                            executablePath,
                            false,
                            affinityMask,
                            selectedProfile,
                          )
                        }
                      />
                      <span className="checkmark circular"></span>
                      Standalone Web Client
                    </label>

                    <label className="checkbox-container">
                      <input
                        type="checkbox"
                        checked={isSteam}
                        onChange={() =>
                          handleSettingsChange(
                            executablePath,
                            true,
                            affinityMask,
                            selectedProfile,
                          )
                        }
                      />
                      <span className="checkmark circular"></span>
                      Steam Client
                    </label>
                  </div>
                </div>
              )}

              <div className="divider" />

              <div className="field">
                <label className="field-label">
                  Nvidia Inspector Profile (.nip)
                </label>
                <select
                  value={selectedProfile}
                  onChange={(e) => {
                    const newProfile = e.target.value;
                    setSelectedProfile(newProfile);
                    handleSettingsChange(
                      executablePath,
                      isSteam,
                      affinityMask,
                      newProfile,
                    );
                  }}
                >
                  <option value="">No profile — skip injection</option>
                  {Object.keys(profiles).map((name) => (
                    <option key={name} value={name}>
                      {name}
                    </option>
                  ))}
                </select>

                {selectedProfile && (
                  <button
                    className={`apply-profile-btn${isApplyingProfile ? " processing" : ""}`}
                    onClick={handleApplyProfile}
                    disabled={isApplyingProfile}
                  >
                    {isApplyingProfile ? "APPLYING..." : "APPLY PROFILE"}
                  </button>
                )}
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
