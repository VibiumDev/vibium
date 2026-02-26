package proxy

// clockScript is the JavaScript that installs a fake clock on `window.__vibiumClock`.
// It overrides Date, setTimeout, setInterval, clearTimeout, clearInterval,
// requestAnimationFrame, cancelAnimationFrame, and performance.now.
const clockScript = `() => {
	if (window.__vibiumClock) return 'already_installed';

	const OrigDate = Date;
	const origSetTimeout = setTimeout;
	const origClearTimeout = clearTimeout;
	const origSetInterval = setInterval;
	const origClearInterval = clearInterval;
	const origRAF = requestAnimationFrame;
	const origCAF = cancelAnimationFrame;
	const origPerfNow = performance.now.bind(performance);

	let currentTime = OrigDate.now();
	let fixedTime = null;
	let paused = false;
	let nextId = 1;
	let resumeTimer = null;
	const timers = new Map();

	// FakeDate class
	class FakeDate extends OrigDate {
		constructor(...args) {
			if (args.length === 0) {
				super(fixedTime !== null ? fixedTime : currentTime);
			} else {
				super(...args);
			}
		}
		static now() {
			return fixedTime !== null ? fixedTime : currentTime;
		}
		static parse(s) { return OrigDate.parse(s); }
		static UTC(...args) { return OrigDate.UTC(...args); }
	}

	function fakeSetTimeout(fn, delay, ...args) {
		if (typeof fn !== 'function') return 0;
		const id = nextId++;
		timers.set(id, {
			callback: fn,
			args: args,
			triggerTime: currentTime + (delay || 0),
			interval: 0,
			type: 'timeout'
		});
		return id;
	}

	function fakeSetInterval(fn, delay, ...args) {
		if (typeof fn !== 'function') return 0;
		const id = nextId++;
		timers.set(id, {
			callback: fn,
			args: args,
			triggerTime: currentTime + (delay || 0),
			interval: delay || 0,
			type: 'interval'
		});
		return id;
	}

	function fakeClearTimeout(id) {
		timers.delete(id);
	}

	function fakeClearInterval(id) {
		timers.delete(id);
	}

	let rafId = 1;
	const rafCallbacks = new Map();

	function fakeRAF(fn) {
		const id = rafId++;
		rafCallbacks.set(id, fn);
		return id;
	}

	function fakeCAF(id) {
		rafCallbacks.delete(id);
	}

	const startPerfTime = origPerfNow();
	const startCurrentTime = currentTime;

	function fakePerfNow() {
		return startPerfTime + (currentTime - startCurrentTime);
	}

	// Install overrides
	window.Date = FakeDate;
	window.setTimeout = fakeSetTimeout;
	window.setInterval = fakeSetInterval;
	window.clearTimeout = fakeClearTimeout;
	window.clearInterval = fakeClearInterval;
	window.requestAnimationFrame = fakeRAF;
	window.cancelAnimationFrame = fakeCAF;
	performance.now = fakePerfNow;

	function getDueTimers(upTo) {
		const due = [];
		for (const [id, t] of timers) {
			if (t.triggerTime <= upTo) {
				due.push([id, t]);
			}
		}
		due.sort((a, b) => a[1].triggerTime - b[1].triggerTime);
		return due;
	}

	function fireRAFs() {
		const cbs = Array.from(rafCallbacks.entries());
		rafCallbacks.clear();
		for (const [, fn] of cbs) {
			try { fn(currentTime); } catch (e) {}
		}
	}

	const clock = {
		fastForward(ms) {
			const target = currentTime + ms;
			currentTime = target;
			// Fire each due timer at most once, no interval rescheduling
			const due = getDueTimers(target);
			for (const [id, t] of due) {
				timers.delete(id);
				try { t.callback(...t.args); } catch (e) {}
			}
			fireRAFs();
		},

		runFor(ms) {
			const target = currentTime + ms;
			while (currentTime < target) {
				// Find the next timer to fire
				let earliest = null;
				let earliestId = null;
				for (const [id, t] of timers) {
					if (t.triggerTime <= target && (!earliest || t.triggerTime < earliest.triggerTime)) {
						earliest = t;
						earliestId = id;
					}
				}
				if (!earliest || earliest.triggerTime > target) {
					currentTime = target;
					break;
				}
				currentTime = earliest.triggerTime;
				if (earliest.type === 'interval' && earliest.interval > 0) {
					earliest.triggerTime = currentTime + earliest.interval;
				} else {
					timers.delete(earliestId);
				}
				try { earliest.callback(...earliest.args); } catch (e) {}
			}
			currentTime = target;
			fireRAFs();
		},

		pauseAt(time) {
			currentTime = time;
			paused = true;
			if (resumeTimer) {
				origClearInterval(resumeTimer);
				resumeTimer = null;
			}
			// Fire due timers
			const due = getDueTimers(time);
			for (const [id, t] of due) {
				timers.delete(id);
				try { t.callback(...t.args); } catch (e) {}
			}
		},

		resume() {
			// No-op if already ticking
			if (resumeTimer) return;
			paused = false;
			let lastReal = OrigDate.now();
			resumeTimer = origSetInterval(() => {
				const now = OrigDate.now();
				const delta = now - lastReal;
				lastReal = now;
				currentTime += delta;
				// Fire due timers
				const due = getDueTimers(currentTime);
				for (const [id, t] of due) {
					if (t.type === 'interval' && t.interval > 0) {
						t.triggerTime = currentTime + t.interval;
					} else {
						timers.delete(id);
					}
					try { t.callback(...t.args); } catch (e) {}
				}
				fireRAFs();
			}, 16);
		},

		setFixedTime(time) {
			fixedTime = time;
		},

		setSystemTime(time) {
			currentTime = time;
			fixedTime = null;
		}
	};

	window.__vibiumClock = clock;
	return 'installed';
}`
