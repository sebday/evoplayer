import QtQuick
import QtTest
import "../../qml/panel/filetree/FiletreeLogic.js" as FiletreeLogic

TestCase {
    name: "FiletreeLogic"
    when: windowShown

    function test_hydrateFolderArtInEntries() {
        var entries = [
            { type: "track", path: "/music/a.mp3", art: "/cache/folder.jpg" },
            { type: "track", path: "/music/b.mp3" },
            { type: "dir", path: "other" }
        ]
        var out = FiletreeLogic.hydrateFolderArtInEntries(entries)
        compare(out[0].art, "/cache/folder.jpg")
        compare(out[1].art, "/cache/folder.jpg")
        compare(out[2].type, "dir")
    }

    function test_folderArtFromSiblings() {
        var children = {
            "grime/2008": [
                { type: "track", path: "/music/grime/2008/a.mp3", art: "/cache/a.jpg" },
                { type: "track", path: "/music/grime/2008/b.mp3" }
            ]
        }
        var art = FiletreeLogic.folderArtFromSiblings(children, "/music/grime/2008/b.mp3")
        compare(art, "/cache/a.jpg")
    }

    function test_patchAlbumScope() {
        var children = {
            "grime/2008": [
                { type: "track", path: "/music/grime/2008/a.mp3" },
                { type: "track", path: "/music/grime/2008/b.mp3" }
            ]
        }
        var rows = [
            { type: "track", path: "/music/grime/2008/a.mp3", track: children["grime/2008"][0] },
            { type: "track", path: "/music/grime/2008/b.mp3", track: children["grime/2008"][1] }
        ]
        var result = FiletreeLogic.patchFiletreeArt(
            children, rows, "/music/grime/2008/a.mp3", "/cache/folder.jpg", "", true)
        compare(result.changed, true)
        compare(result.children["grime/2008"][0].art, "/cache/folder.jpg")
        compare(result.children["grime/2008"][1].art, "/cache/folder.jpg")
    }

    function test_patchTrackScope() {
        var children = {
            "grime/2008": [
                { type: "track", path: "/music/grime/2008/a.mp3" },
                { type: "track", path: "/music/grime/2008/b.mp3" }
            ]
        }
        var rows = [
            { type: "track", path: "/music/grime/2008/a.mp3", track: children["grime/2008"][0] },
            { type: "track", path: "/music/grime/2008/b.mp3", track: children["grime/2008"][1] }
        ]
        var result = FiletreeLogic.patchFiletreeArt(
            children, rows, "/music/grime/2008/a.mp3", "/cache/one.jpg", "", false)
        compare(result.children["grime/2008"][0].art, "/cache/one.jpg")
        compare(result.children["grime/2008"][1].art, undefined)
    }

    function test_rebuildRows() {
        var children = {
            "": [
                { type: "dir", path: "grime", name: "grime", count: 2 }
            ],
            "grime": [
                { type: "track", path: "/music/grime/a.mp3", title: "A" }
            ]
        }
        var expanded = { grime: true }
        var rows = FiletreeLogic.rebuildFiletreeRows(children, expanded)
        compare(rows.length, 2)
        compare(rows[0].type, "dir")
        compare(rows[1].type, "track")
    }

    function test_expansionStateKey() {
        compare(FiletreeLogic.expansionStateKey({ "b/f": true, "a": true }), "a\nb/f")
        compare(FiletreeLogic.expansionStateKey({}), "")
    }

    function test_scrollRestoreY() {
        var map = { "": 0, "grime": 420 }
        compare(FiletreeLogic.scrollRestoreY(map, ""), 0)
        compare(FiletreeLogic.scrollRestoreY(map, "grime"), 420)
        compare(FiletreeLogic.scrollRestoreY(map, "missing"), -1)
    }

    function test_clampContentY() {
        compare(FiletreeLogic.clampContentY(500, 1000, 200), 500)
        compare(FiletreeLogic.clampContentY(900, 1000, 200), 800)
        compare(FiletreeLogic.clampContentY(-1, 1000, 200), 0)
        compare(FiletreeLogic.clampContentY(1200, 1000, 200), 800)
    }

    function test_resolveViewportY_resetToTop() {
        compare(FiletreeLogic.resolveViewportY({
            scrollByKey: {},
            expansionKey: "",
            resetWhenMissing: true
        }), 0)
    }

    function test_resolveViewportY_restoreSaved() {
        compare(FiletreeLogic.resolveViewportY({
            scrollByKey: { "grime": 250 },
            expansionKey: "grime",
            resetWhenMissing: true
        }), 250)
    }

    function test_scrollPerExpansionState() {
        var expCollapsed = { "grime": true }
        var expExpanded = { "grime": true, "grime/2008": true }
        var keyCollapsed = FiletreeLogic.expansionStateKey(expCollapsed)
        var keyExpanded = FiletreeLogic.expansionStateKey(expExpanded)
        var scrollByKey = {}
        scrollByKey[keyCollapsed] = 120
        scrollByKey[keyExpanded] = 480
        compare(FiletreeLogic.scrollRestoreY(scrollByKey, keyCollapsed), 120)
        compare(FiletreeLogic.scrollRestoreY(scrollByKey, keyExpanded), 480)
        compare(FiletreeLogic.scrollRestoreY(scrollByKey, keyCollapsed), 120)
    }

    function test_homeScrollReset() {
        var scrollByKey = {}
        scrollByKey[FiletreeLogic.expansionStateKey({ "drum&bass": true })] = 640
        scrollByKey = {}
        compare(FiletreeLogic.resolveViewportY({
            scrollByKey: scrollByKey,
            expansionKey: "",
            resetWhenMissing: true
        }), 0)
    }

    function test_holdSurvivesContentGrowth() {
        compare(FiletreeLogic.clampContentY(640, 800, 200), 600)
        compare(FiletreeLogic.clampContentY(640, 5000, 200), 640)
    }

    function test_expandScrollPlanHolds() {
        var plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: false,
            holdY: 520,
            restoreY: -1,
            anchorY: 520,
            anchorIndex: 4
        })
        compare(plan.action, "hold")
        compare(plan.y, 520)
    }

    function test_expandDoesNotResetTop() {
        var plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: false,
            holdY: 520,
            restoreY: -1
        })
        compare(plan.action, "hold")
        plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: false,
            holdY: -1,
            restoreY: -1
        })
        compare(plan.action, "resetTop")
    }

    function test_planReflowRestoreAnchorWhileScrolling() {
        var plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: true,
            holdY: -1,
            restoreY: -1,
            anchorY: 640,
            anchorIndex: 12
        })
        compare(plan.action, "restore")
        compare(plan.y, 640)
        compare(plan.anchorIndex, 12)
    }

    function test_planReflowPreservesScrollWithoutResetTop() {
        var plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: true,
            holdY: -1,
            restoreY: -1,
            anchorY: -1,
            anchorIndex: -1
        })
        compare(plan.action, "none")
        verify(plan.action !== "resetTop")
    }

    function test_shouldAcceptFiletreeScrollSave_blocksWhileMoving() {
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ moving: true }))
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ flicking: true }))
        verify(FiletreeLogic.shouldAcceptFiletreeScrollSave({}))
    }

    function test_shouldAcceptFiletreeScrollSave_blocksDuringRestore() {
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ scrollLocked: true }))
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ holdY: 120 }))
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ restoreY: 0 }))
        verify(!FiletreeLogic.shouldAcceptFiletreeScrollSave({ artPatchInFlight: true }))
    }

    function test_shouldDeferFiletreeArtFlush_whileScrolling() {
        verify(FiletreeLogic.shouldDeferFiletreeArtFlush({ userScrolling: true }))
        verify(FiletreeLogic.shouldDeferFiletreeArtFlush({ scrollLocked: true }))
        verify(!FiletreeLogic.shouldDeferFiletreeArtFlush({}))
    }

    function test_artPatchCanUpdateChildrenWithoutRowSwap() {
        var children = {
            "grime/2008": [
                { type: "track", path: "/music/grime/2008/a.mp3" },
                { type: "track", path: "/music/grime/2008/b.mp3" }
            ]
        }
        var rows = [
            { type: "track", path: "/music/grime/2008/a.mp3", track: children["grime/2008"][0] },
            { type: "track", path: "/music/grime/2008/b.mp3", track: children["grime/2008"][1] }
        ]
        var result = FiletreeLogic.patchFiletreeArtBatch(children, rows, {
            "/music/grime/2008/a.mp3": { art: "/cache/one.jpg", thumb: "", albumScope: false }
        })
        compare(result.changed, true)
        compare(result.children["grime/2008"][0].art, "/cache/one.jpg")
        compare(rows[0].track.art, undefined)
    }

    function test_fastScrollDoesNotOverwriteSavedPosition() {
        var key = FiletreeLogic.expansionStateKey({ grime: true })
        var scrollByKey = {}
        scrollByKey[key] = 640
        compare(FiletreeLogic.scrollRestoreY(scrollByKey, key), 640)
        if (FiletreeLogic.shouldAcceptFiletreeScrollSave({ moving: true, flicking: true }))
            scrollByKey[key] = 0
        compare(FiletreeLogic.scrollRestoreY(scrollByKey, key), 640)
    }

    function test_warmPathsForFolder() {
        var entries = [
            { type: "track", path: "/music/grime/2009/a.mp3" },
            { type: "track", path: "/music/grime/2009/b.mp3" },
            { type: "dir", path: "other" }
        ]
        var paths = FiletreeLogic.warmPathsForFolder(entries, 50)
        compare(paths.length, 1)
        compare(paths[0], "/music/grime/2009/a.mp3")
    }

    function test_warmPathsSkipsComplete() {
        var entries = [
            { type: "track", path: "/a.mp3", art: "/art.jpg", thumb: "/thumb.jpg" },
            { type: "track", path: "/b.mp3" }
        ]
        var paths = FiletreeLogic.warmPathsForFolder(entries, 50)
        compare(paths.length, 1)
        compare(paths[0], "/b.mp3")
    }

    function test_folderArtFromEntries() {
        var shared = FiletreeLogic.folderArtFromEntries([
            { type: "track", path: "/a.mp3", art: "/cache/folder.jpg" },
            { type: "track", path: "/b.mp3" }
        ])
        compare(shared.art, "/cache/folder.jpg")
        compare(shared.thumb, "")
    }

    function test_parseBrowseQueueTracks() {
        var tracks = FiletreeLogic.parseBrowseQueueTracks(
            JSON.stringify({ tracks: [{ path: "/a.mp3" }, { path: "/b.mp3" }] }))
        compare(tracks.length, 2)
        compare(tracks[0].path, "/a.mp3")
        var fromPaths = FiletreeLogic.parseBrowseQueueTracks(
            JSON.stringify({ paths: ["/a.mp3", "/b.mp3"] }))
        compare(fromPaths.length, 2)
        compare(fromPaths[1].title, "b.mp3")
    }

    function test_appendTracksUnique() {
        var result = FiletreeLogic.appendTracksUnique(
            [{ path: "/a.mp3" }],
            [{ path: "/a.mp3" }, { path: "/b.mp3" }])
        compare(result.added, 1)
        compare(result.tracks.length, 2)
        compare(result.tracks[1].path, "/b.mp3")
    }

    function test_pathsFromTracks() {
        var paths = FiletreeLogic.pathsFromTracks([
            { path: "/a.mp3" },
            { path: "/b.mp3" }
        ])
        compare(paths.length, 2)
        compare(paths[0], "/a.mp3")
    }

    function test_collectFolderTracksFromChildren() {
        var children = {
            "dubstep/vinyl/afbar": [
                { type: "track", path: "/music/loose.mp3", title: "Loose" },
                { type: "dir", path: "dubstep/vinyl/afbar/album" }
            ],
            "dubstep/vinyl/afbar/album": [
                { type: "track", path: "/music/nested.mp3", title: "Nested" }
            ]
        }
        var tracks = FiletreeLogic.collectFolderTracksFromChildren(
            children, "dubstep/vinyl/afbar")
        compare(tracks.length, 2)
        compare(tracks[0].path, "/music/loose.mp3")
        compare(tracks[1].path, "/music/nested.mp3")
    }

    function test_folderActionReserve() {
        compare(FiletreeLogic.folderActionReserve(2, false) >= 68, true)
        compare(FiletreeLogic.folderActionReserve(4, true) >= 140, true)
        compare(FiletreeLogic.folderActionReserve(4, false) >= 120, true)
    }

    function test_normalizeFolderEntry() {
        var entry = FiletreeLogic.normalizeFolderEntry("dubstep/vinyl/afbar")
        compare(entry.type, "dir")
        compare(entry.path, "dubstep/vinyl/afbar")
        compare(entry.name, "afbar")
        verify(!FiletreeLogic.normalizeFolderEntry({ type: "track", path: "/a.mp3" }))
    }
}
