(function () {
  const PREFIX = "music-server-v1";

  function readJSON(key, fallback) {
    try {
      const raw = localStorage.getItem(PREFIX + ":" + key);
      if (!raw) return fallback;
      return JSON.parse(raw);
    } catch (err) {
      return fallback;
    }
  }

  function writeJSON(key, value) {
    localStorage.setItem(PREFIX + ":" + key, JSON.stringify(value));
  }

  window.PlayerStorage = {
    getFavorites() {
      return new Set(readJSON("favorites", []));
    },
    setFavorites(set) {
      writeJSON("favorites", Array.from(set));
    },
    getView() {
      const raw = readJSON("view", {});
      return {
        groupKey: raw.groupKey || "all",
        sortKey: raw.sortKey || "default",
        sortDir: raw.sortDir === "desc" ? "desc" : "asc",
      };
    },
    setView(view) {
      writeJSON("view", {
        groupKey: view.groupKey || "all",
        sortKey: view.sortKey || "default",
        sortDir: view.sortDir === "desc" ? "desc" : "asc",
      });
    },
    getPlay() {
      const raw = readJSON("play", {});
      return {
        groupKey: raw.groupKey || "all",
        sortKey: raw.sortKey || "default",
        sortDir: raw.sortDir === "desc" ? "desc" : "asc",
        playMode: raw.playMode || "list:all",
      };
    },
    setPlay(play) {
      writeJSON("play", {
        groupKey: play.groupKey || "all",
        sortKey: play.sortKey || "default",
        sortDir: play.sortDir === "desc" ? "desc" : "asc",
        playMode: play.playMode || "list:all",
      });
    },
    getPlaying() {
      const raw = readJSON("playing", {});
      return {
        trackKey: raw.trackKey || "",
      };
    },
    setPlaying(playing) {
      writeJSON("playing", {
        trackKey: playing.trackKey || "",
      });
    },
  };
})();
