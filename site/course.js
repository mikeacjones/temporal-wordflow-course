// Docsify plugins for the course site:
// - show each lesson's build.md inside the lesson, under "Build it"
// - save "Check it" ticks in the browser and mark finished lessons in the sidebar
// - point the Web app / Temporal UI links at the right forwarded port

(function () {
  const BUILD_LINK = /^Follow \[`build\.md`\]\(build\.md\)\.\s*$/m;
  const STORE = "wordflow:";

  // Codespaces forward port N as https://<name>-N.app.github.dev.
  function portURL(port) {
    const m = location.hostname.match(/^(.+)-\d+\.(.+)$/);
    if (m && /github\.dev$|githubpreview\.dev$/.test(m[2])) {
      return `${location.protocol}//${m[1]}-${port}.${m[2]}/`;
    }
    return `${location.protocol}//${location.hostname}:${port}/`;
  }

  document.querySelectorAll(".app-links a[data-port]").forEach((a) => {    a.href = portURL(a.dataset.port);
  });

  // Drop build.md's title and nest its sections under "Build it".
  function nestBuild(md) {
    let fenced = false;
    let titleDropped = false;
    return md
      .split("\n")
      .flatMap((line) => {
        if (/^\s*(```|~~~)/.test(line)) fenced = !fenced;
        if (fenced) return [line];
        if (!titleDropped && /^# /.test(line)) {
          titleDropped = true;
          return [];
        }
        return [/^#{1,5} /.test(line) ? "#" + line : line];
      })
      .join("\n");
  }

  function load(key) {
    try {
      return JSON.parse(localStorage.getItem(STORE + key)) || {};
    } catch {
      return {};
    }
  }

  function save(key, value) {
    localStorage.setItem(STORE + key, JSON.stringify(value));
  }

  function markSidebar() {
    const progress = load("progress");
    document.querySelectorAll(".sidebar-nav a[href^='#/']").forEach((a) => {
      const href = a.getAttribute("href");
      if (href.includes("?id=")) return;
      const p = progress[decodeURIComponent(href.slice(1))];
      a.classList.toggle("done", !!p && p.total > 0 && p.done === p.total);
    });
  }

  function checklist(vm) {
    const path = vm.route.path;
    const boxes = [...document.querySelectorAll(".markdown-section input[type=checkbox]")];
    const ticks = load("ticks");
    const saved = ticks[path] || [];

    const update = () => {
      ticks[path] = boxes.map((b) => b.checked);
      save("ticks", ticks);
      const progress = load("progress");
      progress[path] = { done: boxes.filter((b) => b.checked).length, total: boxes.length };
      save("progress", progress);
      markSidebar();
    };

    boxes.forEach((box, i) => {
      box.disabled = false;
      box.checked = !!saved[i];
      box.closest("li")?.classList.toggle("checked", box.checked);
      box.addEventListener("change", () => {
        box.closest("li")?.classList.toggle("checked", box.checked);
        update();
      });
    });
    if (boxes.length) update();
    else markSidebar();
  }

  window.$docsify.plugins.push(function (hook, vm) {
    hook.mounted(function () {
      document.querySelector(".sidebar .app-name")?.after(document.querySelector(".app-links"));
    });

    hook.beforeEach(function (content, next) {
      if (!BUILD_LINK.test(content)) return next(content);
      const dir = vm.route.file.replace(/[^/]*$/, "");
      fetch("/" + dir + "build.md", { cache: "no-store" })
        .then((r) => (r.ok ? r.text() : Promise.reject(r.status)))
        .then((build) => next(content.replace(BUILD_LINK, () => nestBuild(build))))
        .catch(() =>
          next(content.replace(BUILD_LINK, "> The build steps come from your language branch. Open this course from a language branch, such as `go`.")),
        );
    });

    hook.doneEach(function () {
      checklist(vm);
    });
  });
})();
