(async function () {
  const storage = window.PlayerStorage;
  const favoriteBtn = document.getElementById("toggleFavorite");
  const categoryFilterEl = document.getElementById("categoryFilter");
  const viewToggleEl = document.getElementById("viewToggle");
  const categoryListEl = document.getElementById("categoryList");
  const builtinListEl = document.getElementById("builtinList");
  const playAllBtn = document.getElementById("playAllBtn");
  const trackTbodyEl = document.getElementById("trackTbody");
  const trackSearchInputEl = document.getElementById("trackSearchInput");
  const searchPrevBtn = document.getElementById("searchPrevBtn");
  const searchNextBtn = document.getElementById("searchNextBtn");
  const searchCountEl = document.getElementById("searchCount");
  const browseTitleEl = document.getElementById("browseTitle");
  const sortHeaderEls = Array.from(document.querySelectorAll(".th-sort"));

  const resp = await fetch(window.MUSIC_JSON_URL);
  if (!resp.ok) {
    return;
  }

  const music = await resp.json();
  const favorites = storage.getFavorites();
  const savedView = storage.getView();
  const savedPlay = storage.getPlay();
  const savedPlaying = storage.getPlaying();

  const allTracks = music.map((item, idx) => normalizeTrack(item, idx));
  const groups = buildGroups(allTracks, favorites);

  const builtInRows = [
    { key: "all", label: `全部 (${groups.all.length})` },
    { key: "favorite", label: `收藏 ★ (${groups.favorite.length})` },
  ];
  const categoryRows = {
    album: Object.keys(groups.album).sort().map((name) => ({
      key: `album:${name}`,
      name,
      count: groups.album[name].length,
    })),
    artist: Object.keys(groups.artist).sort().map((name) => ({
      key: `artist:${name}`,
      name,
      count: groups.artist[name].length,
    })),
  };

  const ap = new APlayer({
    container: document.getElementById("aplayer"),
    lrcType: 3,
    audio: [],
    listFolded: false,
    order: getSavedOrder(savedPlay.playMode),
    loop: getSavedLoop(savedPlay.playMode),
  });

  let currentGroupKey = isValidGroupKey(savedView.groupKey, groups) ? savedView.groupKey : "all";
  let currentPlayGroupKey = "all";
  let currentPlaySortKey = "default";
  let currentPlaySortDir = "asc";
  let currentPlayList = [];
  let lastKnownPlayMode = "";
  let playBtnSyncQueued = false;
  let importSwitchToken = 0;
  let currentBrowseList = resolveList(groups, currentGroupKey);
  let currentCategoryView = currentGroupKey.startsWith("artist:") ? "artist" : "album";
  let currentSortKey = isValidSortKey(savedView.sortKey) ? savedView.sortKey : "default";
  let currentSortDir = savedView.sortDir === "desc" ? "desc" : "asc";
  let searchQuery = "";
  let searchMatches = [];
  let currentSearchIndex = -1;

  renderGroupList(builtinListEl, builtInRows);
  renderCategoryList();
  updateActiveGroup();
  updateToggleButtons();
  renderSortHeaders();
  renderTrackTable();

  restorePlayAndPlaying(savedPlay, savedPlaying);

  builtinListEl.addEventListener("click", (ev) => {
    const li = ev.target.closest("li[data-key]");
    if (!li) return;
    selectGroup(li.dataset.key);
  });

  categoryListEl.addEventListener("click", (ev) => {
    const li = ev.target.closest("li[data-key]");
    if (!li) return;
    selectGroup(li.dataset.key);
  });

  viewToggleEl.addEventListener("click", (ev) => {
    const btn = ev.target.closest("button[data-view]");
    if (!btn) return;
    currentCategoryView = btn.dataset.view;
    renderCategoryList();
    updateToggleButtons();
    updateActiveGroup();
  });

  categoryFilterEl.addEventListener("input", () => {
    renderCategoryList();
    updateActiveGroup();
  });

  sortHeaderEls.forEach((btn) => {
    btn.addEventListener("click", () => {
      const key = btn.dataset.sort;
      if (!key) return;

      if (currentSortKey === key) {
        currentSortDir = currentSortDir === "asc" ? "desc" : "asc";
      } else {
        currentSortKey = key;
        currentSortDir = "asc";
      }
      renderSortHeaders();
      renderTrackTable();
      saveViewState();
    });
  });

  playAllBtn.addEventListener("click", () => {
    const sorted = getSortedBrowseList();
    if (!sorted.length) return;

    if (canReuseCurrentPlayList(sorted)) {
      ap.list.switch(0);
      ap.play();
    } else {
      adoptViewAsPlayContext();
      currentPlayList = sorted;
      importToPlayerAndPlay(sorted, 0);
    }
    savePlayState();
    savePlayingState();
  });

  trackTbodyEl.addEventListener("click", (ev) => {
    const btn = ev.target.closest("button[data-action]");
    if (!btn) return;
    const key = btn.dataset.key;
    if (!key) return;
    const action = btn.dataset.action;
    if (action === "play") {
      playTrackByKey(key);
      return;
    }
    if (action === "favorite") {
      toggleFavoriteByKey(key);
    }
  });

  if (trackSearchInputEl) {
    trackSearchInputEl.addEventListener("keydown", (ev) => {
      if (ev.key !== "Enter") return;
      ev.preventDefault();
      const q = trackSearchInputEl.value.trim();
      if (!q) {
        clearTrackSearch();
        return;
      }
      if (q === searchQuery && searchMatches.length) {
        moveSearchMatch(1);
        return;
      }
      runTrackSearch(q);
    });

    trackSearchInputEl.addEventListener("input", () => {
      const q = trackSearchInputEl.value.trim();
      if (q.length > 2) {
        runTrackSearch(q);
        return;
      }
      clearTrackSearch();
    });

    trackSearchInputEl.addEventListener("blur", () => {
      const q = trackSearchInputEl.value.trim();
      if (!q) {
        clearTrackSearch();
        return;
      }
      if (q === searchQuery) {
        return;
      }
      runTrackSearch(q);
    });
  }

  if (searchPrevBtn) {
    searchPrevBtn.addEventListener("click", () => {
      moveSearchMatch(-1);
    });
  }

  if (searchNextBtn) {
    searchNextBtn.addEventListener("click", () => {
      moveSearchMatch(1);
    });
  }

  updateSearchNavButtons();

  favoriteBtn.addEventListener("click", () => {
    const current = ap.list.audios[ap.list.index];
    if (!current) return;
    toggleFavoriteByKey(current._trackKey);
  });

  ap.on("listswitch", handlePlayerTrackChange);
  ap.on("ended", handlePlayerTrackChange);
  ap.on("play", handlePlayerTrackChange);
  bindPlayModePersistence();

  function handlePlayerTrackChange() {
    savePlayingState();
    queuePlayButtonSync();
  }

  function queuePlayButtonSync() {
    if (playBtnSyncQueued) return;
    playBtnSyncQueued = true;
    requestAnimationFrame(() => {
      playBtnSyncQueued = false;
      syncPlayButtonState();
    });
  }

  function syncPlayButtonState() {
    const playingKey = getCurrentPlayingTrackKey();
    const buttons = trackTbodyEl.querySelectorAll('button[data-action="play"]');
    buttons.forEach((btn) => {
      const isActive = !!playingKey && btn.dataset.key === playingKey;
      btn.classList.toggle("is-active", isActive);
    });
  }

  function runTrackSearch(rawQuery) {
    const q = String(rawQuery || "").trim();
    if (!q) {
      clearTrackSearch();
      return;
    }
    searchQuery = q;
    searchMatches = collectSearchMatches(searchQuery);
    applySearchHitClasses();
    if (!searchMatches.length) {
      currentSearchIndex = -1;
      updateSearchNavButtons();
      updateSearchCounter();
      return;
    }
    currentSearchIndex = 0;
    applyCurrentSearchMarker();
    scrollMatchIntoView(currentSearchIndex);
    updateSearchNavButtons();
    updateSearchCounter();
  }

  function clearTrackSearch() {
    searchQuery = "";
    searchMatches = [];
    currentSearchIndex = -1;
    clearSearchClasses();
    updateSearchNavButtons();
    updateSearchCounter();
  }

  function collectSearchMatches(query) {
    const norm = normalizeSearchText(query);
    if (!norm) return [];
    const rows = Array.from(trackTbodyEl.querySelectorAll("tr[data-search-text]"));
    return rows.filter((row) => row.dataset.searchText.includes(norm));
  }

  function applySearchHitClasses() {
    clearSearchClasses();
    searchMatches.forEach((row) => {
      row.classList.add("search-hit");
    });
  }

  function applyCurrentSearchMarker() {
    trackTbodyEl.querySelectorAll("tr.search-current").forEach((row) => {
      row.classList.remove("search-current");
    });
    if (currentSearchIndex < 0 || currentSearchIndex >= searchMatches.length) return;
    const row = searchMatches[currentSearchIndex];
    row.classList.add("search-current");
  }

  function clearSearchClasses() {
    trackTbodyEl.querySelectorAll("tr.search-hit, tr.search-current").forEach((row) => {
      row.classList.remove("search-hit", "search-current");
    });
  }

  function moveSearchMatch(step) {
    if (!searchMatches.length) return;
    const total = searchMatches.length;
    currentSearchIndex = (currentSearchIndex + step + total) % total;
    applyCurrentSearchMarker();
    scrollMatchIntoView(currentSearchIndex);
    updateSearchCounter();
  }

  function scrollMatchIntoView(idx) {
    if (idx < 0 || idx >= searchMatches.length) return;
    const row = searchMatches[idx];
    row.scrollIntoView({ block: "center", behavior: "smooth" });
  }

  function updateSearchNavButtons() {
    const disabled = searchMatches.length === 0;
    if (searchPrevBtn) searchPrevBtn.disabled = disabled;
    if (searchNextBtn) searchNextBtn.disabled = disabled;
  }

  function updateSearchCounter() {
    if (!searchCountEl) return;
    const total = searchMatches.length;
    const current = total > 0 && currentSearchIndex >= 0 ? currentSearchIndex + 1 : 0;
    searchCountEl.textContent = `${current}/${total}`;
  }

  function reapplyTrackSearchAfterRender() {
    if (!searchQuery) {
      clearSearchClasses();
      updateSearchNavButtons();
      updateSearchCounter();
      return;
    }

    const prevKey = currentSearchIndex >= 0 && currentSearchIndex < searchMatches.length
      ? searchMatches[currentSearchIndex].dataset.trackKey
      : "";

    searchMatches = collectSearchMatches(searchQuery);
    applySearchHitClasses();

    if (!searchMatches.length) {
      currentSearchIndex = -1;
      updateSearchNavButtons();
      updateSearchCounter();
      return;
    }

    if (prevKey) {
      const idx = searchMatches.findIndex((row) => row.dataset.trackKey === prevKey);
      currentSearchIndex = idx >= 0 ? idx : 0;
    } else {
      currentSearchIndex = 0;
    }

    applyCurrentSearchMarker();
    updateSearchNavButtons();
    updateSearchCounter();
  }

  function normalizeSearchText(text) {
    return String(text || "").trim().toLowerCase();
  }

  function bindPlayModePersistence() {
    ap.container.addEventListener("click", (ev) => {
      // Do not rely on target element type/class (e.g. SVG path).
      // Compare mode snapshot after click and persist only when changed.
      setTimeout(() => {
        if (!currentPlayList.length) return;
        const mode = getCurrentPlayMode();
        if (mode === lastKnownPlayMode) return;
        savePlayState();
      }, 0);
    });
  }

  function restorePlayAndPlaying(playState, playingState) {
    const resolvedGroup = isValidGroupKey(playState.groupKey, groups) ? playState.groupKey : "all";
    const resolvedSortKey = isValidSortKey(playState.sortKey) ? playState.sortKey : "default";
    const resolvedSortDir = playState.sortDir === "desc" ? "desc" : "asc";
    const list = sortTracks(resolveList(groups, resolvedGroup), resolvedSortKey, resolvedSortDir);
    if (!list.length) return;

    currentPlayList = list;
    currentPlayGroupKey = resolvedGroup;
    currentPlaySortKey = resolvedSortKey;
    currentPlaySortDir = resolvedSortDir;

    let idx = 0;
    if (playingState.trackKey) {
      const found = list.findIndex((it) => it._trackKey === playingState.trackKey);
      if (found >= 0) idx = found;
    }

    loadPlayerListWithScapegoat(list, idx, false);
    lastKnownPlayMode = getCurrentPlayMode();
  }

  function saveViewState() {
    storage.setView({
      groupKey: currentGroupKey,
      sortKey: currentSortKey,
      sortDir: currentSortDir,
    });
  }

  function savePlayState() {
    const playMode = getCurrentPlayMode();
    lastKnownPlayMode = playMode;
    storage.setPlay({
      groupKey: currentPlayGroupKey,
      sortKey: currentPlaySortKey,
      sortDir: currentPlaySortDir,
      playMode,
    });
  }

  function savePlayingState() {
    storage.setPlaying({
      trackKey: getCurrentPlayingTrackKey(),
    });
  }

  function getCurrentPlayingTrackKey() {
    const current = ap.list.audios[ap.list.index];
    return current ? current._trackKey : "";
  }

  function selectGroup(key) {
    currentGroupKey = key;
    if (key.startsWith("artist:")) {
      currentCategoryView = "artist";
      updateToggleButtons();
      renderCategoryList();
    } else if (key.startsWith("album:")) {
      currentCategoryView = "album";
      updateToggleButtons();
      renderCategoryList();
    }
    currentBrowseList = resolveList(groups, key);
    renderTrackTable();
    updateActiveGroup();
    saveViewState();
  }

  function playTrackByKey(trackKey) {
    const sorted = getSortedBrowseList();
    const idx = sorted.findIndex((it) => it._trackKey === trackKey);
    if (idx < 0) return;

    if (canReuseCurrentPlayList(sorted)) {
      ap.list.switch(idx);
      ap.play();
    } else {
      adoptViewAsPlayContext();
      currentPlayList = sorted;
      importToPlayerAndPlay(sorted, idx);
    }
    savePlayState();
    savePlayingState();
  }

  function adoptViewAsPlayContext() {
    currentPlayGroupKey = currentGroupKey;
    currentPlaySortKey = currentSortKey;
    currentPlaySortDir = currentSortDir;
  }

  function importToPlayerAndPlay(list, idx) {
    loadPlayerListWithScapegoat(list, idx, true);
  }

  function loadPlayerListWithScapegoat(list, idx, shouldPlay) {
    const token = ++importSwitchToken;
    ap.list.clear();

    const prepared = prepareListForFirstLrcRace(list, idx);
    ap.list.add(prepared.items);
    ap.list.switch(prepared.switchIndex);

    if (shouldPlay) {
      ap.play();
    }

    if (prepared.hasScapegoat) {
      setTimeout(() => {
        if (token !== importSwitchToken) return;
        ap.list.remove(0);
      }, 200);
    }
  }

  function prepareListForFirstLrcRace(list, idx) {
    const shouldUseScapegoat = idx > 0 && list[0] && list[0].lrc;
    if (!shouldUseScapegoat) {
      return {
        items: list,
        switchIndex: idx,
        hasScapegoat: false,
      };
    }

    const scapegoat = {
      name: "列表加载中...",
      artist: " ",
      url: "data:,",
      lrc: "data:,",
    };
    return {
      items: [scapegoat, ...list],
      switchIndex: idx + 1,
      hasScapegoat: true,
    };
  }

  function canReuseCurrentPlayList(targetList) {
    if (!isPlayContextSameAsView()) return false;

    // Favorite list can change when star state changes, so keep strict list checks there.
    if (currentPlayGroupKey !== "favorite" && currentGroupKey !== "favorite") {
      return true;
    }

    if (!isSameTrackList(currentPlayList, targetList)) return false;
    return true;
  }

  function isPlayContextSameAsView() {
    return currentPlayGroupKey === currentGroupKey
      && currentPlaySortKey === currentSortKey
      && currentPlaySortDir === currentSortDir;
  }

  function isSameTrackList(a, b) {
    if (!Array.isArray(a) || !Array.isArray(b)) return false;
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i += 1) {
      if (a[i]._trackKey !== b[i]._trackKey) {
        return false;
      }
    }
    return true;
  }

  function toggleFavoriteByKey(trackKey) {
    if (favorites.has(trackKey)) {
      favorites.delete(trackKey);
    } else {
      favorites.add(trackKey);
    }

    storage.setFavorites(favorites);
    groups.favorite = groups.all.filter((item) => favorites.has(item._trackKey));
    builtInRows[1].label = `收藏 ★ (${groups.favorite.length})`;
    renderGroupList(builtinListEl, builtInRows);

    currentBrowseList = resolveList(groups, currentGroupKey);
    renderTrackTable();
    updateActiveGroup();

    if (currentPlayGroupKey === "favorite") {
      const keepKey = ap.list.audios[ap.list.index]?._trackKey || "";
      currentPlayList = sortTracks(groups.favorite, currentPlaySortKey, currentPlaySortDir);
      refreshPlayList(currentPlayList, keepKey);
      savePlayState();
      savePlayingState();
    }
  }

  function refreshPlayList(nextList, keepTrackKey) {
    currentPlayList = nextList;
    if (!currentPlayList.length) {
      ap.list.clear();
      return;
    }

    ap.list.clear();
    currentPlayList.forEach((item) => ap.list.add(item));

    let idx = 0;
    if (keepTrackKey) {
      const found = currentPlayList.findIndex((it) => it._trackKey === keepTrackKey);
      if (found >= 0) idx = found;
    }
    ap.list.switch(idx);
  }

  function applyMode(mode) {
    if (typeof mode === "string" && mode.includes(":")) {
      const [orderRaw, loopRaw] = mode.split(":", 2);
      const order = normalizeOrder(orderRaw);
      const loop = normalizeLoop(loopRaw);
      if (ap.options) {
        ap.options.order = order;
        ap.options.loop = loop;
      }
      return;
    }

    if (["list", "single", "random", "none"].includes(mode)) {
      ap.list.mode = mode;
    }
  }

  function getCurrentPlayMode() {
    const order = normalizeOrder(ap.options && ap.options.order);
    const loop = normalizeLoop(ap.options && ap.options.loop);
    return `${order}:${loop}`;
  }

  function normalizeOrder(order) {
    return ["list", "random"].includes(order) ? order : "list";
  }

  function normalizeLoop(loop) {
    return ["all", "one", "none"].includes(loop) ? loop : "all";
  }

  function getSavedOrder(playMode) {
    if (typeof playMode === "string" && playMode.includes(":")) {
      return normalizeOrder(playMode.split(":", 2)[0]);
    }
    return "list";
  }

  function getSavedLoop(playMode) {
    if (typeof playMode === "string" && playMode.includes(":")) {
      return normalizeLoop(playMode.split(":", 2)[1]);
    }
    return "all";
  }

  function renderCategoryList() {
    const keyword = categoryFilterEl.value.trim().toLowerCase();
    const rows = categoryRows[currentCategoryView].filter((row) => {
      if (!keyword) return true;
      return row.name.toLowerCase().includes(keyword);
    }).map((row) => ({
      key: row.key,
      label: `${row.name} (${row.count})`,
    }));

    renderGroupList(categoryListEl, rows);
  }

  function renderTrackTable() {
    browseTitleEl.textContent = `列表内容 - ${groupLabel(currentGroupKey)}`;
    const list = getSortedBrowseList();
    const playingKey = getCurrentPlayingTrackKey();
    trackTbodyEl.innerHTML = list.map((item) => {
      const favClass = favorites.has(item._trackKey) ? "small-btn icon-btn is-active" : "small-btn icon-btn";
      const playClass = item._trackKey === playingKey ? "small-btn icon-btn is-active" : "small-btn icon-btn";
      const downloadName = `${item._artist} - ${item.name}`;
      const lrcBtn = item.lrc
        ? `<a class="small-btn icon-btn" href="${escapeHTML(item.lrc)}" download title="下载lrc">⤓lrc</a>`
        : `<span class="small-btn icon-btn is-disabled" title="lrc不可用">⤓lrc</span>`;
        return `<tr data-track-key="${escapeHTML(item._trackKey)}" data-search-text="${escapeHTML(normalizeSearchText(`${item._title} ${item._artist} ${item._album}`))}">
  <td>${escapeHTML(item.name)}</td>
  <td>${escapeHTML(item._artist)}</td>
  <td>${escapeHTML(item._album)}</td>
  <td>${escapeHTML(item._durationText)}</td>
  <td class="track-op">
    <button class="${favClass}" type="button" data-action="favorite" data-key="${escapeHTML(item._trackKey)}" title="收藏">★</button>
    <button class="${playClass} icon-play" type="button" data-action="play" data-key="${escapeHTML(item._trackKey)}" title="播放">▶</button>
    <a class="small-btn icon-btn icon-download" href="${escapeHTML(item.url)}" download="${escapeHTML(downloadName)}" title="下载">⤓</a>
    ${lrcBtn}
  </td>
</tr>`;
    }).join("");
    reapplyTrackSearchAfterRender();
  }

  function updateToggleButtons() {
    viewToggleEl.querySelectorAll("button[data-view]").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.view === currentCategoryView);
    });
  }

  function updateActiveGroup() {
    document.querySelectorAll(".group-list li").forEach((x) => x.classList.remove("active"));
    const escaped = cssEscape(currentGroupKey);
    const target = document.querySelector(`.group-list li[data-key="${escaped}"]`);
    if (target) target.classList.add("active");
  }

  function normalizeTrack(item, index) {
    const title = item.title || item.name || "Unknown";
    const artist = item.artist || "Unknown";
    const album = item.album || "未知专辑";
    const durationSec = Number.isFinite(item.length) ? Math.max(0, Math.floor(item.length)) : null;
    const _trackKey = `${item.name || title}-${artist}`;
    return {
      name: title,
      artist: `${artist} · ${album}`,
      url: item.url,
      cover: item.cover || "",
      lrc: item.lrc || "",
      _trackKey,
      _album: album,
      _artist: artist,
      _title: title,
      _durationSec: durationSec,
      _durationText: formatDuration(durationSec),
      _index: index,
    };
  }

  function buildGroups(list, favSet) {
    const album = {};
    const artist = {};

    list.forEach((t) => {
      if (!album[t._album]) album[t._album] = [];
      album[t._album].push(t);
      if (!artist[t._artist]) artist[t._artist] = [];
      artist[t._artist].push(t);
    });

    return {
      all: list,
      favorite: list.filter((t) => favSet.has(t._trackKey)),
      album,
      artist,
    };
  }

  function resolveList(groupObj, key) {
    if (key === "all") return groupObj.all;
    if (key === "favorite") return groupObj.favorite;
    if (key.startsWith("album:")) return groupObj.album[key.slice(6)] || [];
    if (key.startsWith("artist:")) return groupObj.artist[key.slice(7)] || [];
    return groupObj.all;
  }

  function isValidGroupKey(key, groupObj) {
    if (!key || typeof key !== "string") return false;
    if (key === "all" || key === "favorite") return true;
    if (key.startsWith("album:")) return Array.isArray(groupObj.album[key.slice(6)]);
    if (key.startsWith("artist:")) return Array.isArray(groupObj.artist[key.slice(7)]);
    return false;
  }

  function isValidSortKey(key) {
    return ["default", "title", "artist", "album", "duration"].includes(key);
  }

  function getSortedBrowseList() {
    return sortTracks(currentBrowseList, currentSortKey, currentSortDir);
  }

  function sortTracks(list, sortBy, dir) {
    const out = [...list];
    out.sort((a, b) => {
      if (sortBy === "default") {
        return a._index - b._index;
      }
      if (sortBy === "artist") {
        const x = a._artist.localeCompare(b._artist, "zh-Hans-CN");
        if (x !== 0) return x;
        return a._title.localeCompare(b._title, "zh-Hans-CN");
      }
      if (sortBy === "album") {
        const x = a._album.localeCompare(b._album, "zh-Hans-CN");
        if (x !== 0) return x;
        return a._title.localeCompare(b._title, "zh-Hans-CN");
      }
      if (sortBy === "title") {
        return a._title.localeCompare(b._title, "zh-Hans-CN");
      }
      if (sortBy === "duration") {
        const ad = Number.isFinite(a._durationSec) ? a._durationSec : Number.MAX_SAFE_INTEGER;
        const bd = Number.isFinite(b._durationSec) ? b._durationSec : Number.MAX_SAFE_INTEGER;
        if (ad !== bd) {
          return ad - bd;
        }
        return a._title.localeCompare(b._title, "zh-Hans-CN");
      }
      return 0;
    });

    if (dir === "desc") {
      out.reverse();
    }
    return out;
  }

  function renderSortHeaders() {
    sortHeaderEls.forEach((btn) => {
      const key = btn.dataset.sort;
      const isActive = key === currentSortKey;
      btn.classList.toggle("active", isActive);
      const base = sortText(key);
      const suffix = isActive ? (currentSortDir === "asc" ? " ↑" : " ↓") : "";
      btn.textContent = base + suffix;
    });
  }

  function sortText(sortBy) {
    switch (sortBy) {
      case "artist":
        return "艺术家";
      case "album":
        return "专辑";
      case "title":
        return "标题";
      case "duration":
        return "时长";
      default:
        return sortBy;
    }
  }

  function formatDuration(totalSec) {
    if (!Number.isFinite(totalSec)) {
      return "--:--";
    }
    const sec = Math.max(0, Number(totalSec) || 0);
    const mm = Math.floor(sec / 60);
    const ss = sec % 60;
    return `${String(mm).padStart(2, "0")}:${String(ss).padStart(2, "0")}`;
  }

  function groupLabel(key) {
    if (key === "all") return "全部";
    if (key === "favorite") return "收藏 ★";
    if (key.startsWith("album:")) return `专辑: ${key.slice(6)}`;
    if (key.startsWith("artist:")) return `艺术家: ${key.slice(7)}`;
    return key;
  }

  function renderGroupList(el, rows) {
    el.innerHTML = rows
      .map((r) => `<li data-key="${escapeHTML(r.key)}">${escapeHTML(r.label)}</li>`)
      .join("");
  }

  function cssEscape(value) {
    if (window.CSS && typeof window.CSS.escape === "function") {
      return window.CSS.escape(value);
    }
    return String(value).replaceAll('"', '\\"');
  }

  function escapeHTML(s) {
    return String(s)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }
})();
