// Shared test harness for the stale-issue closer tests (`stale-eng-issues.test.js` and
// `stale-fleetie-issues.test.js`). Provides time helpers and mocks for the `github`, `context`,
// and `core` objects that `actions/github-script` passes to the scripts. Issue factories stay in
// the individual test files because each wrapper has a different notion of a default-eligible issue.

'use strict';

const DAY_MS = 24 * 60 * 60 * 1000;
const daysAgoIso = (days) => new Date(Date.now() - days * DAY_MS).toISOString();

// `at` (ms-since-epoch) takes precedence over `daysAgo` so tests aligning an event to a specific
// updated_at moment can share the same instant rather than two independent Date.now() reads.
function makeStaleLabelEvent({ daysAgo = 100, at } = {}) {
  const created_at = at != null ? new Date(at).toISOString() : daysAgoIso(daysAgo);
  return { event: 'labeled', label: { name: 'stale' }, created_at };
}

function makeContext() {
  return { repo: { owner: 'o', repo: 'r' } };
}

function makeCore() {
  const infos = [];
  const warnings = [];
  const summaryCalls = [];
  const summary = new Proxy(
    {},
    {
      get(_target, prop) {
        if (prop === 'write') {
          return async () => {
            summaryCalls.push({ method: 'write' });
          };
        }
        return (...args) => {
          summaryCalls.push({ method: String(prop), args });
          return summary;
        };
      },
    },
  );
  return {
    info: (msg) => infos.push(msg),
    warning: (msg) => warnings.push(msg),
    summary,
    _captured: { infos, warnings, summaryCalls },
  };
}

// `issuesByPage`: array of pages (each page is an array of issues) returned by paginate.iterator.
// `eventsByIssue`: map from issue.number -> array of event objects returned by paginate(listEvents).
// `failOn`: optional fault-injection { createComment, addLabels, update, removeLabel, listEvents }
//           values can be 'always', an integer (fail until Nth call), or { status: 404 }.
function makeGithub({ issuesByPage = [], eventsByIssue = {}, failOn = {} } = {}) {
  const createCommentCalls = [];
  const addLabelsCalls = [];
  const removeLabelCalls = [];
  const updateCalls = [];
  const listEventsCalls = [];

  const counters = {};
  const shouldFail = (op) => {
    const cfg = failOn[op];
    if (!cfg) return null;
    counters[op] = (counters[op] || 0) + 1;
    if (cfg === 'always') return new Error(`${op} simulated failure`);
    if (typeof cfg === 'number' && counters[op] <= cfg) return new Error(`${op} simulated failure`);
    if (typeof cfg === 'object' && cfg.status && counters[op] === 1) {
      const err = new Error(`${op} simulated failure status ${cfg.status}`);
      err.status = cfg.status;
      return err;
    }
    return null;
  };

  const paginate = async (endpoint, params) => {
    if (endpoint === 'listEvents-sentinel') {
      listEventsCalls.push(params);
      const err = shouldFail('listEvents');
      if (err) throw err;
      return eventsByIssue[params.issue_number] || [];
    }
    if (endpoint === 'listForRepo-sentinel') {
      return issuesByPage.flat();
    }
    return [];
  };
  paginate.iterator = async function* iterator(endpoint) {
    if (endpoint === 'listForRepo-sentinel') {
      for (const page of issuesByPage) yield { data: page };
    }
  };

  return {
    paginate,
    rest: {
      issues: {
        listForRepo: 'listForRepo-sentinel',
        listEvents: 'listEvents-sentinel',
        createComment: async (params) => {
          const err = shouldFail('createComment');
          if (err) throw err;
          createCommentCalls.push(params);
        },
        addLabels: async (params) => {
          const err = shouldFail('addLabels');
          if (err) throw err;
          addLabelsCalls.push(params);
        },
        removeLabel: async (params) => {
          const err = shouldFail('removeLabel');
          if (err) throw err;
          removeLabelCalls.push(params);
        },
        update: async (params) => {
          const err = shouldFail('update');
          if (err) throw err;
          updateCalls.push(params);
        },
      },
    },
    _captured: { createCommentCalls, addLabelsCalls, removeLabelCalls, updateCalls, listEventsCalls },
  };
}

module.exports = { DAY_MS, daysAgoIso, makeStaleLabelEvent, makeContext, makeCore, makeGithub };
