.pragma library

function trackPathForEntry(entry) {
    if (!entry)
        return ""
    return String(entry.path || (entry.track && entry.track.path) || "")
}

function folderArtFromEntries(entries) {
    entries = entries || []
    var folderArt = ""
    var folderThumb = ""
    var i, entry, art, thumb
    for (i = 0; i < entries.length; i++) {
        entry = entries[i]
        if (!entry || entry.type !== "track")
            continue
        art = String(entry.art || "")
        thumb = String(entry.thumb || "")
        if (!folderArt && art.charAt(0) === "/")
            folderArt = art
        if (!folderThumb && thumb.charAt(0) === "/")
            folderThumb = thumb
    }
    return { art: folderArt, thumb: folderThumb }
}

function hydrateFolderArtInEntries(entries) {
    if (!entries || !entries.length)
        return entries
    var shared = folderArtFromEntries(entries)
    var folderArt = shared.art
    var folderThumb = shared.thumb
    if (!folderArt && !folderThumb)
        return entries
    var out = []
    var i
    for (i = 0; i < entries.length; i++) {
        if (entries[i].type !== "track") {
            out.push(entries[i])
            continue
        }
        var patch = {}
        if (folderArt && !entries[i].art)
            patch.art = folderArt
        if (folderThumb && !entries[i].thumb)
            patch.thumb = folderThumb
        out.push(Object.keys(patch).length ? Object.assign({}, entries[i], patch) : entries[i])
    }
    return out
}

function folderArtFromSiblings(children, trackPath) {
    trackPath = String(trackPath || "")
    if (!trackPath || !children)
        return ""
    for (var key in children) {
        var kids = children[key]
        var inFolder = false
        var i
        for (i = 0; i < kids.length; i++) {
            if (kids[i].type !== "track")
                continue
            var p = String(kids[i].path || (kids[i].track && kids[i].track.path) || "")
            if (p === trackPath) {
                inFolder = true
                break
            }
        }
        if (!inFolder)
            continue
        for (i = 0; i < kids.length; i++) {
            if (kids[i].type !== "track")
                continue
            var s = kids[i].track || kids[i]
            var thumb = String(s.thumb || "")
            if (thumb.charAt(0) === "/")
                return thumb
            var art = String(s.art || "")
            if (art.charAt(0) === "/")
                return art
        }
        return ""
    }
    return ""
}

function trackMatchSpec(trackPath, spec) {
    trackPath = String(trackPath || "")
    if (!trackPath || !spec)
        return false
    if (trackPath === spec.trackPath)
        return true
    if (!spec.albumScope || !spec.dir)
        return false
    if (trackPath.indexOf(spec.dir) !== 0)
        return false
    return trackPath.indexOf("/", spec.dir.length) < 0
}

function buildMatchSpecs(queue) {
    var specs = []
    for (var path in queue) {
        var item = queue[path]
        var trackPath = String(path || "")
        var slash = trackPath.lastIndexOf("/")
        specs.push({
            trackPath: trackPath,
            art: String(item.art || ""),
            thumb: String(item.thumb || ""),
            albumScope: !!item.albumScope,
            dir: slash >= 0 ? trackPath.substring(0, slash + 1) : ""
        })
    }
    return specs
}

function patchKidFields(kid, fields) {
    if (!Object.keys(fields).length)
        return kid
    var patched = Object.assign({}, kid, fields)
    if (kid.track)
        patched.track = Object.assign({}, kid.track, fields)
    return patched
}

function fieldsForPath(path, specs) {
    var art = ""
    var thumb = ""
    for (var i = 0; i < specs.length; i++) {
        if (!trackMatchSpec(path, specs[i]))
            continue
        if (specs[i].art)
            art = specs[i].art
        if (specs[i].thumb)
            thumb = specs[i].thumb
    }
    var fields = {}
    if (art)
        fields.art = art
    if (thumb)
        fields.thumb = thumb
    return fields
}

function patchFiletreeArt(children, rows, trackPath, art, thumb, albumScope) {
    var queue = {}
    queue[String(trackPath || "")] = {
        art: String(art || ""),
        thumb: thumb !== undefined ? String(thumb || "") : "",
        albumScope: !!albumScope
    }
    return patchFiletreeArtBatch(children, rows, queue)
}

function patchFiletreeArtBatch(children, rows, queue) {
    children = children || {}
    rows = rows || []
    var specs = buildMatchSpecs(queue)
    if (!specs.length)
        return { children: children, rows: rows, changed: false }

    var nextChildren = {}
    var changed = false
    for (var key in children) {
        var kids = children[key]
        var nextKids = []
        for (var i = 0; i < kids.length; i++) {
            var kid = kids[i]
            if (kid.type === "track") {
                var fields = fieldsForPath(trackPathForEntry(kid), specs)
                if (Object.keys(fields).length) {
                    nextKids.push(patchKidFields(kid, fields))
                    changed = true
                } else {
                    nextKids.push(kid)
                }
            } else {
                nextKids.push(kid)
            }
        }
        nextChildren[key] = nextKids
    }
    if (!changed)
        return { children: children, rows: rows, changed: false }

    var nextRows = []
    for (var r = 0; r < rows.length; r++) {
        var row = rows[r]
        if (row.type === "track") {
            var rowFields = fieldsForPath(trackPathForEntry(row.track || row), specs)
            if (Object.keys(rowFields).length) {
                var patchedRow = patchKidFields(row.track || row, rowFields)
                nextRows.push(Object.assign({}, row, {
                    track: patchedRow,
                    path: String(patchedRow.path || row.path || "")
                }))
            } else {
                nextRows.push(row)
            }
        } else {
            nextRows.push(row)
        }
    }
    return { children: nextChildren, rows: nextRows, changed: true }
}

function rebuildFiletreeRows(children, expanded) {
    children = children || {}
    expanded = expanded || {}
    var rows = []
    function appendChildren(parentPath, depth) {
        var key = String(parentPath || "")
        var kids = children[key]
        if (!kids)
            return
        for (var i = 0; i < kids.length; i++) {
            var kid = kids[i]
            if (kid.type === "track") {
                var trackPath = String(kid.path || (kid.track && kid.track.path) || "")
                rows.push({
                    type: "track",
                    folderPath: key,
                    path: trackPath,
                    name: String(kid.title || kid.name || trackPath.split("/").pop() || ""),
                    artist: String(kid.artist || ""),
                    track: kid,
                    depth: depth
                })
                continue
            }
            var path = String(kid.path || "")
            var isExpanded = !!expanded[path]
            rows.push({
                type: "dir",
                path: path,
                name: String(kid.name || path.split("/").pop() || path),
                depth: depth,
                expanded: isExpanded,
                count: kid.count
            })
            if (isExpanded)
                appendChildren(path, depth + 1)
        }
    }
    appendChildren("", 0)
    return rows
}

function expansionStateKey(expanded) {
    expanded = expanded || {}
    var paths = []
    for (var p in expanded) {
        if (expanded[p])
            paths.push(String(p))
    }
    paths.sort()
    return paths.join("\n")
}

function scrollRestoreY(scrollByKey, key) {
    scrollByKey = scrollByKey || {}
    var y = scrollByKey[String(key !== undefined ? key : "")]
    if (y !== undefined && y !== null && y >= 0)
        return Number(y)
    return -1
}

function clampContentY(contentY, contentHeight, viewportHeight) {
    var y = Number(contentY)
    if (isNaN(y) || y < 0)
        y = 0
    var maxY = Math.max(0, Number(contentHeight) - Number(viewportHeight))
    if (isNaN(maxY) || maxY < 0)
        maxY = 0
    return Math.min(y, maxY)
}

function resolveViewportY(options) {
    options = options || {}
    var key = options.expansionKey !== undefined ? String(options.expansionKey) : ""
    var restoreY = options.explicitRestoreY
    if (restoreY === undefined || restoreY === null)
        restoreY = scrollRestoreY(options.scrollByKey, key)
    if (restoreY >= 0)
        return Number(restoreY)
    if (options.resetWhenMissing)
        return 0
    return -1
}

function shouldDeferFiletreeArtFlush(options) {
    options = options || {}
    return !!(options.scrollLocked || options.userScrolling)
}

function shouldAcceptFiletreeScrollSave(options) {
    options = options || {}
    if (options.scrollLocked)
        return false
    if (Number(options.holdY) >= 0)
        return false
    if (Number(options.restoreY) >= 0)
        return false
    if (options.moving || options.flicking)
        return false
    if (options.artPatchInFlight)
        return false
    return true
}

function planFiletreeReflowScroll(options) {
    options = options || {}
    var holdY = Number(options.holdY)
    if (holdY >= 0)
        return { action: "hold", y: holdY }
    if (options.preserveScroll
            && (Number(options.anchorY) >= 0 || Number(options.anchorIndex) >= 0)) {
        return {
            action: "restore",
            y: Number(options.anchorY),
            anchorIndex: Number(options.anchorIndex)
        }
    }
    if (!options.preserveScroll && holdY < 0 && Number(options.restoreY) < 0)
        return { action: "resetTop" }
    return { action: "none" }
}

function warmPathsForFolder(entries, limit) {
    entries = entries || []
    limit = limit > 0 ? limit : 50
    var paths = []
    var i, entry, art, thumb
    for (i = 0; i < entries.length && paths.length < limit; i++) {
        entry = entries[i]
        if (!entry || entry.type !== "track")
            continue
        var trackPath = trackPathForEntry(entry)
        if (!trackPath)
            continue
        art = String(entry.art || "")
        thumb = String(entry.thumb || "")
        if (art.charAt(0) === "/" && thumb.charAt(0) === "/")
            continue
        paths.push(trackPath)
    }
    if (paths.length > 1)
        return [paths[0]]
    return paths
}

function folderTracksComplete(children, folderMeta, folderPath) {
    folderPath = String(folderPath || "")
    if (!folderPath)
        return false
    var meta = folderMeta && folderMeta[folderPath]
    if (!meta || !meta.total)
        return false
    return Number(meta.loaded) >= Number(meta.total)
}

function collectFolderTracksFromChildren(children, folderPath) {
    children = children || {}
    folderPath = String(folderPath || "")
    var tracks = []
    function walk(key) {
        var kids = children[key] || []
        var i, kid
        for (i = 0; i < kids.length; i++) {
            kid = kids[i]
            if (kid.type === "track")
                tracks.push(kid.track || kid)
            else if (kid.type === "dir")
                walk(String(kid.path || ""))
        }
    }
    walk(folderPath)
    return tracks
}

function parseBrowseQueueTracks(text) {
    try {
        var data = JSON.parse(String(text || "{}"))
        if (Array.isArray(data.paths) && data.paths.length)
            return tracksFromPaths(data.paths)
        return Array.isArray(data.tracks) ? data.tracks : []
    } catch (e) {
        return []
    }
}

function pathsFromTracks(tracks) {
    tracks = tracks || []
    var paths = []
    var i, track, path
    for (i = 0; i < tracks.length; i++) {
        track = tracks[i]
        path = track ? String(track.path || "") : ""
        if (path)
            paths.push(path)
    }
    return paths
}

function tracksFromPaths(paths) {
    paths = paths || []
    var tracks = []
    var i, path
    for (i = 0; i < paths.length; i++) {
        path = String(paths[i] || "")
        if (!path)
            continue
        tracks.push({ path: path, type: "track" })
    }
    return tracks
}

function appendTracksUnique(existing, incoming) {
    existing = existing || []
    incoming = incoming || []
    var seen = {}
    var i, path
    for (i = 0; i < existing.length; i++) {
        path = trackPathForEntry(existing[i])
        if (path)
            seen[path] = true
    }
    var merged = existing.slice()
    var added = 0
    for (i = 0; i < incoming.length; i++) {
        path = trackPathForEntry(incoming[i])
        if (!path || seen[path])
            continue
        seen[path] = true
        merged.push(incoming[i])
        added++
    }
    return { tracks: merged, added: added }
}

function folderActionReserve(iconCount, showCount) {
    iconCount = Number(iconCount) || 0
    var reserve = iconCount * 28 + Math.max(0, iconCount - 1) * 6 + 10
    if (showCount)
        reserve += 28
    return reserve
}

function normalizeFolderEntry(entry) {
    if (!entry)
        return null
    if (typeof entry === "string") {
        var p = String(entry || "").trim()
        if (!p)
            return null
        return { type: "dir", path: p, name: p.split("/").pop() || p }
    }
    if (entry.type !== "dir")
        return null
    var path = String(entry.path || "").trim()
    if (!path)
        return null
    return {
        type: "dir",
        path: path,
        name: String(entry.name || path.split("/").pop() || path)
    }
}
