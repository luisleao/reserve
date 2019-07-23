// Copyright 2019 The Reserve Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

window.__reserve_hooks_by_extension = {
  html: f => new_f => {
    // The current page, minus any query string or hash.
    let curpage = new URL(location.pathname, location.href).href;
    let target = f.replace(/index\.html$/, '');
    if (curpage == target)
      location.reload(true);
  },
};

(() => {
  const defaultHook = f => new_f => {
    let handled = false;
    for (let el of document.querySelectorAll('link')) {
      if (el.rel == "x-reserve-ignore") {
        const re = new RegExp(el.dataset.expr);
        if (re.test(f))
          handled = true;
        continue;
      }
      if (el.href != f && el.dataset.ohref != f)
        continue;
      if (!el.dataset.ohref)
        el.dataset.ohref = el.href;
      el.href = new_f;
      handled = true;
    }
    return handled;
  };
  const hooks = {};

  const cacheBustQuery = () => `?cache_bust=${+new Date}`;
  const es = new EventSource("/.reserve/changes");
  es.addEventListener('change', e => {
    const target = new URL(e.data, location.href).href;
    const cacheBustedTarget = target + cacheBustQuery();

    if (!(target in hooks)) {
      const ext = target.split('/').pop().split('.').pop();
      const genHook = window.__reserve_hooks_by_extension[ext];
      hooks[target] = genHook ? genHook(target) : () => Promise.resolve();
    }
    Promise.resolve()
      .then(() => hooks[target](cacheBustedTarget))
      .then(handled => handled || defaultHook(target)(cacheBustedTarget))
      .then(handled => handled || location.reload(true));
  });

  let wasOpen = false;
  es.addEventListener('open', e => {
    if (wasOpen)
      location.reload(true);
    wasOpen = true;
  });

  let stdin = new EventSource("/.reserve/stdin");
  stdin.addEventListener("line", e => {
    const ev = new CustomEvent('stdin');
    ev.data = e.data;
    window.dispatchEvent(ev);
  });
})();
