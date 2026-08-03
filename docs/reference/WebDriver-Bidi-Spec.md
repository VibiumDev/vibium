<pre class=metadata>
Title: WebDriver BiDi
Shortname: webdriver-bidi
Level: None
Status: ED
Group: browser-testing-tools
URL: https://w3c.github.io/webdriver-bidi/
TR: https://www.w3.org/TR/webdriver-bidi/
Repository: w3c/webdriver-bidi
Editor: James Graham, Mozilla https://www.mozilla.org, w3cid 40334
Editor: Alex Rudenko, Google https://www.google.com, w3cid 141088
Editor: Maksim Sadym, Google https://www.google.com, w3cid 128970
Abstract: This document defines the BiDirectional WebDriver Protocol, a mechanism for remote control of user agents.
Boilerplate: conformance no
Complain About: accidental-2119 yes, missing-example-ids yes
Default Ref Status: current
Indent: 2
Implementation Report: https://wpt.fyi/results/webdriver/tests/bidi
Test Suite: https://github.com/web-platform-tests/wpt/tree/master/webdriver/tests/bidi
!Channel: <a href="https://www.w3.org/wiki/IRC">#webdriver on irc.w3.org</a>
!Wiki: <a href="https://www.w3.org/wiki/WebDriver">W3C WebDriver Wiki</a>
</pre>

<pre class=anchors>
spec: RFC5280; urlPrefix: https://datatracker.ietf.org/doc/html/rfc5280
  type: dfn
    text: Basic Certificate Processing; url: section-6.1.3
spec: RFC6455; urlPrefix: https://datatracker.ietf.org/doc/html/rfc6455
  type: dfn
    text: WebSocket URI; url: section-3
    text: Establishes a WebSocket Connection; url: section-4.1
    text: Server-Side Requirements; url: section-4.2
    text: Reading the Client's Opening Handshake; url: section-4.2.1
    text: %x1 denotes a text frame; url: section-5.2
    text: Send a WebSocket Message; url: section-6.1
    text: A WebSocket Message Has Been Received; url: section-6.2
    text: Start The WebSocket Closing Handshake; url: section-7.1.2
    text: The WebSocket Closing Handshake is Started; url: section-7.1.3
    text: The WebSocket Connection is Closed; url: section-7.1.4
    text: Fail the WebSocket Connection; url: section-7.1.7
    text: Status Codes; url: section-7.4
    text: Handling Errors in UTF-8-Encoded Data; url: section-8.1
spec: RFC8610; urlPrefix: https://datatracker.ietf.org/doc/html/rfc8610
  type: dfn
    text: match a CDDL specification; url: appendix-C
spec: RFC6265
  type: dfn
    text: Cookie; url: https://httpwg.org/specs/rfc6265.html
    text: Cookie store; url: https://httpwg.org/specs/rfc6265.html#storage-model
spec: RFC6265bis; urlPrefix: https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-rfc6265bis-20.html
  type: dfn
    text: Cookie Lifetime Limits; url: #cookie-lifetime-limits
    text: Lax; url: section-4.1.2.7
    text: Strict; url: section-4.1.2.7
    text: Default; url: section-5.6.7.2
spec: WEBDRIVER; urlPrefix: https://w3c.github.io/webdriver/
  type: dfn
    text: WebDriver new session algorithm; url: dfn-webdriver-new-session-algorithms
    text: accept insecure TLS; url: dfn-accept-insecure-tls
    text: actions; url: actions
    text: actions options; url: dfn-actions-options
    text: active sessions; url: dfn-active-sessions
    text: additional WebDriver capability; url: dfn-additional-webdriver-capability
    text: additional capability deserialization algorithm; url: dfn-additional-capability-deserialization-algorithm
    text: capability name; url: dfn-capability-name
    text: close the session; url: dfn-close-the-session
    text: cookie domain; url: dfn-cookie-domain
    text: cookie expiry time; url: dfn-cookie-expiry-time
    text: cookie HTTP only; url: dfn-cookie-http-only
    text: cookie name; url: dfn-cookie-name
    text: cookie path; url: dfn-cookie-path
    text: cookie same site; url: dfn-cookie-same-site
    text: cookie secure only; url: dfn-cookie-secure-only
    text: cookie value; url: dfn-cookie-value
    text: create a cookie; url: dfn-creating-a-cookie
    text: create a session; url: dfn-create-a-session
    text: dispatch actions; url: dfn-dispatch-actions
    text: dispatch tick actions; url: dfn-dispatch-tick-actions
    text: draw a bounding box from the framebuffer; url: dfn-draw-a-bounding-box-from-the-framebuffer
    text: endpoint node; url: dfn-endpoint-node
    text: error code; url: dfn-error-code
    text: error; url: errors
    text: extract an action sequence; url: dfn-extract-an-action-sequence
    text: get a node; url: dfn-get-a-node
    text: get element origin; url: dfn-get-element-origin
    text: get or create a node reference; url: dfn-get-or-create-a-node-reference
    text: get the input state; url: dfn-get-the-input-state
    text: getting a property; url: dfn-getting-properties
    text: http session; url: dfn-http-session
    text: input cancel list; url: dfn-input-cancel-list
    text: intermediary node; url: dfn-intermediary-nodes
    text: invalid argument; url: dfn-invalid-argument
    text: invalid selector; url: dfn-invalid-selector
    text: invalid session id; url: dfn-invalid-session-id
    text: is element origin; url: dfn-is-element-origin
    text: local end; url: dfn-local-ends
    text: matched capability serialization algorithm; url: dfn-matched-capability-serialization-algorithm
    text: maximum active sessions; url: dfn-maximum-active-sessions
    text: no such alert; url: dfn-no-such-alert
    text: no such element; url: dfn-no-such-element
    text: no such frame; url: dfn-no-such-frame
    text: parse a page range; url: dfn-parse-a-page-range
    text: handler; for: prompt handler configuration; url: dfn-handler
    text: process capabilities; url: dfn-capabilities-processing
    text: proxy configuration; url: dfn-proxy-configuration
    text: readiness state; url: dfn-readiness-state
    text: remote end steps; url: dfn-remote-end-steps
    text: remote end; url: dfn-remote-ends
    text: reset the input state; url: dfn-reset-the-input-state
    text: scroll into view; url: dfn-scrolls-into-view
    text: session ID; url: dfn-session-id
    text: session not created; url: dfn-session-not-created
    text: session; url: dfn-webdriver-session
    text: set a property; url: dfn-set-a-property
    text: success; url: dfn-success
    text: table for cookie conversion; url: dfn-table-for-cookie-conversion
    text: try; url: dfn-try
    text: trying; url: dfn-try
    text: unable to capture screen; url: dfn-unable-to-capture-screen
    text: unknown command; url: dfn-unknown-command
    text: unknown error; url: dfn-unknown-error
    text: user prompt handler; url: dfn-user-prompt-handler
    text: unsupported operation; url: dfn-unsupported-operation
    text: web element reference; url: dfn-web-element-reference
    text: webdriver-active flag; url: dfn-webdriver-active-flag
    text: window handle; url: dfn-window-handles
spec: CONSOLE; urlPrefix: https://console.spec.whatwg.org
  type: dfn
    text: formatter; url: formatter
    text: formatting specifier; url: formatting-specifiers
    text: printer; url: printer
spec: ECMASCRIPT-I18N; urlPrefix: https://tc39.es/ecma402/
  type: dfn
    text: DefaultLocale; url: #sec-defaultlocale
    text: IsStructurallyValidLanguageTag; url: #sec-isstructurallyvalidlanguagetag
spec: ECMASCRIPT; urlPrefix: https://tc39.es/ecma262/
  type: dfn
    text: Array; url: sec-array-objects
    text: AvailableNamedTimeZoneIdentifiers; url: sec-availablenamedtimezoneidentifiers
    text: Await; url: await
    text: BigInt; url: sec-bigint-constructor
    text: Call; url: sec-call
    text: Completion Record; url: sec-completion-record-specification-type
    text: Construct; url: sec-construct
    text: CreateArrayFromList; url: sec-createarrayfromlist
    text: CreateArrayIterator; url: sec-createarrayiterator
    text: CreateBuiltinFunction; url: sec-createbuiltinfunction
    text: CreateListFromArrayLike; url: sec-createlistfromarraylike
    text: CreateMapIterator; url: sec-createmapiterator
    text: CreateSetIterator; url: sec-createsetiterator
    text: Date Time String Format; url: sec-date-time-string-format
    text: Date.prototype.toISOString; url: sec-date.prototype.toisostring
    text: Date; url: sec-date-constructor
    text: EnumerableOwnPropertyNames; url: sec-enumerableownpropertynames
    text: Get; url: sec-get-o-p
    text: GetIterator; url: sec-getiterator
    text: HasProperty; url: sec-hasproperty
    text: IsArray; url: sec-isarray
    text: IsCallable; url: sec-iscallable
    text: IsPromise; url: sec-ispromise
    text: IsRegExp; url: sec-isregexp
    text: IsTimeZoneOffsetString; url: sec-istimezoneoffsetstring
    text: IteratorToList; url: sec-iteratortolist
    text: LengthOfArrayLike; url: sec-lengthofarraylike
    text: Map; url: #sec-map-iterable; for: constructor
    text: Number; url: sec-number-constructor
    text: Object.fromEntries; url: sec-object.fromentries
    text: Object; url: sec-object-objects
    text: RegExp; url: sec-regexp-pattern-flags
    text: ScriptEvaluation; url: sec-runtime-semantics-scriptevaluation
    text: Set object; url: sec-set-objects
    text: String; url: sec-string-constructor
    text: StringToBigInt; url: sec-stringtobigint
    text: StringToNumber; url: sec-stringtonumber
    text: SystemTimeZoneIdentifier; url: sec-systemtimezoneidentifier
    text: ToString; url: sec-tostring
    text: Type; url: sec-ecmascript-data-types-and-values
    text: abrupt completion; url: sec-completion-record-specification-type
    text: boolean; url: sec-terms-and-definitions-boolean-value
    text: current realm record; url: current-realm
    text: internal slot; url: sec-object-internal-methods-and-internal-slots
    text: null; url: sec-null-value
    text: primitive ECMAScript value; url: sec-primitive-value
    text: realm; url: sec-code-realms
    text: running execution context; url: running-execution-context
    text: throw completion; url: sec-completion-record-specification-type
    text: test; url: #sec-regexp.prototype.test
    text: time value; url: sec-time-values-and-time-range
    text: undefined; url: sec-undefined-value
spec: FETCH; urlPrefix: https://fetch.spec.whatwg.org/
  type: dfn
    text: response; url: #concept-response
spec: GEOMETRY; urlPrefix: https://drafts.fxtf.org/geometry/
  type: dfn
    text: rectangle; url: rectangle
    text: x coordinate; url: rectangle-x-coordinate
    text: y coordinate; url: rectangle-y-coordinate
    text: width dimension; url: rectangle-width-dimension
    text: height dimension; url: rectangle-height-dimension
spec: GEOLOCATION; urlPrefix: https://www.w3.org/TR/geolocation/
  type: dfn
    text: GeolocationPositionError; url: #dom-geolocationpositionerror
    text: POSITION_UNAVAILABLE; url: #dom-geolocationpositionerror-position_unavailable
    text: set emulated position data; url: #dfn-set-emulated-position-data
spec: SCREEN-ORIENTATION; urlPrefix: https://www.w3.org/TR/screen-orientation
  type: dfn
    text: current orientation angle; url: #dfn-current-orientation-angle
    text: current orientation type; url: #dfn-current-orientation-type
    text: screen orientation; url: #dom-screenorientation
    text: screen orientation change steps; url: #dfn-screen-orientation-change-steps
    text: screen orientation values lists; url: #dfn-screen-orientation-values-lists
spec: HTML; urlPrefix: https://html.spec.whatwg.org/multipage/
  type: dfn
    text: 2D context creation algorithm; url: canvas.html#2d-context-creation-algorithm
    text: 2D; url: canvas.html#concept-canvas-2d
    text: a serialization of the bitmap as a file; url: canvas.html#a-serialisation-of-the-bitmap-as-a-file
    text: activation notification; url: interaction.html#activation-notification
    text: active window; url: document-sequences.html#nav-window
    text: alert; url: timers-and-user-prompts.html#dom-alert
    text: associated `Document`; url: nav-history-apis.html#concept-document-window
    text: close; url: document-sequences.html#close-a-top-level-traversable
    text: disabled; url: form-control-infrastructure.html#concept-fe-disabled
    text: File Upload state; url: input.html#file-upload-state-(type=file)
    text: confirm; url: timers-and-user-prompts.html#dom-confirm
    text: context mode; url: /canvas.html#offscreencanvas-context-mode
    text: create a classic script; url: webappapis.html#creating-a-classic-script
    text: create a new browsing context; url: browsers.html#creating-a-new-browsing-context
    text: create a new top-level traversable; url: document-sequences.html#creating-a-new-top-level-traversable
    text: default script fetch options; url: webappapis.html#default-script-fetch-options
    text: default view; url: nav-history-apis.html#dom-document-defaultview
    text: descendant navigables; url: document-sequences.html#descendant-navigables
    text: environment settings object's Realm; url: webappapis.html#environment-settings-object's-realm
    text: focused area of the document; url: interaction.html#focused-area-of-the-document
    text: getting all used history steps; url:browsing-the-web.html#getting-all-used-history-steps
    text: hidden; url: document-sequences.html#system-visibility-state
    text: history handling behavior; url: browsing-the-web.html#history-handling-behavior
    text: innerText getter steps; url:dom.html#dom-innertext
    text: input type; url: input.html#dom-input-type
    text: navigable; for:window; url: nav-history-apis.html#window-navigable
    text: navigables; url: document-sequences.html#navigables
    text: navigation id; url: browsing-the-web.html#navigation-id
    text: ongoing-navigation; url: browsing-the-web.html#ongoing-navigation
    text: origin-clean; url: canvas.html#concept-canvas-origin-clean
    text: parent; for:navigable; url: document-sequences.html#nav-parent
    text: prompt to unload; url: browsing-the-web.html#prompt-to-unload-a-document
    text: prompt; url: timers-and-user-prompts.html#dom-prompt
    text: report an error; url: webappapis.html#report-the-error
    text: run the animation frame callbacks; url: imagebitmap-and-animations.html#run-the-animation-frame-callbacks
    text: same origin domain; url: browsers.html#same-origin-domain
    text: select an image source from a source set; url: images.html#select-an-image-source-from-a-source-set
    text: selected files; url: input.html#concept-input-type-file-selected
    text: session history entry; url: browsing-the-web.html#session-history-entry
    text: session history traversal queue; url: document-sequences.html#tn-session-history-traversal-queue
    text: session history; url: history.html#session-history
    text: set up a window environment settings object; url: nav-history-apis.html#set-up-a-window-environment-settings-object
    text: set up a worker environment settings object; url: workers.html#set-up-a-worker-environment-settings-object
    text: set up a worklet environment settings object; url: worklets.html#set-up-a-worklet-environment-settings-object
    text: shared worker; url: workers.html#shared-workers
    text: system visibility state; url: document-sequences.html#system-visibility-state
    text: traversable navigable; url:document-sequences.html#traversable-navigable
    text: traverse the history by a delta; url: browsing-the-web.html#traverse-the-history-by-a-delta
    text: update the file selection; url: input.html#update-the-file-selection
    text: visible; url: document-sequences.html#system-visibility-state
    text: worker event loop; url: webappapis.html#worker-event-loop-2
    text: worklet global scopes; url:worklets.html#concept-document-worklet-global-scopes
spec: INFRA; urlPrefix: https://infra.spec.whatwg.org/
  type: dfn
    text: break; url: iteration-break
    text: convert a JSON-derived JavaScript value to an Infra value; url: convert-a-json-derived-javascript-value-to-an-infra-value
spec: RESOURCE-TIMING; urlPrefix: https://w3c.github.io/resource-timing/
  type: dfn
    text: convert fetch timestamp; url: dfn-convert-fetch-timestamp
spec: HR-TIME; urlPrefix: https://w3c.github.io/hr-time/
  type: dfn
    text: get time origin timestamp; url: get-time-origin-timestamp
spec: RFC4648; urlPrefix: https://datatracker.ietf.org/doc/html/rfc4648
  type: dfn
    text: Base64 Encode; url: section-4
spec: CSS-OVERFLOW-3; urlPrefix: https://drafts.csswg.org/css-overflow-3/
  type: dfn
    text: classic scrollbars; url: #classic-scrollbars
    text: overlay scrollbars; url: #overlay-scrollbars
spec: CSS-VALUES-3; urlPrefix: https://drafts.csswg.org/css-values-3/
  type: dfn
    text: absolute lengths; url: #absolute-lengths
spec: CSSOM-VIEW; urlPrefix: https://drafts.csswg.org/cssom-view/
  type: dfn
    text: CSS pixel; url: #dom-window-devicepixelratio
    text: evaluate media queries and report changes; url: #evaluate-media-queries-and-report-changes
    text: layout viewport; url: #layout-viewport
    text: scroll height; url: #dom-element-scrollheight
    text: scroll width; url: #dom-element-scrollwidth
    text: visual viewport page left; url: #dom-visualviewport-pageleft
    text: visual viewport page top; url: #dom-visualviewport-pagetop
    text: visual viewport; url: #visual-viewport
    text: web-exposed available screen area; url: #web-exposed-available-screen-area
    text: web-exposed screen area; url: #web-exposed-screen-area
spec: DOM; urlPrefix: https://dom.spec.whatwg.org/
  type: dfn
    text: root; url: #concept-tree-root
    text: document element; url: #ref-for-dom-document-documentelement
    text: evaluate; url: #dom-xpathevaluatorbase-evaluate
    text: nodes; url: #concept-node
    text: ORDERED_NODE_SNAPSHOT_TYPE; url: #dom-xpathresult-ordered_node_snapshot_type
    text: snapshotItem; url: #dom-xpathresult-snapshotitem
spec: FULLSCREEN; urlPrefix: https://fullscreen.spec.whatwg.org/
  type: dfn
    text: fullscreen an element; url: #fullscreen-an-element
    text: fullscreen is supported; url: #fullscreen-is-supported
    text: fully exit fullscreen; url: #fully-exit-fullscreen
spec: SELECTORS4; urlPrefix: https://drafts.csswg.org/selectors-4/
  type: dfn
    text: match a selector against a tree; url: #match-a-selector-against-a-tree
    text: parse a selector; url: #parse-a-selector
    text: scoping root; url: #scoping-root
spec: WEB-IDL; urlPrefix: https://webidl.spec.whatwg.org/
  type: dfn
    text: DOMException; url: #idl-DOMException
    text: SyntaxError; url: #syntaxerror
spec: UNICODE; urlPrefix: https://www.unicode.org/versions/Unicode15.0.0/
  type: dfn
    text: Unicode Default Case Conversion algorithm; url: ch03.pdf#G34944
    text: toUppercase; url: ch03.pdf#G34078
spec: ACCNAME; urlPrefix:https://www.w3.org/TR/accname-1.2
  type: dfn
    text: accessible name; url: /#dfn-accessible-name
spec: CORE-AAM; urlPrefix:https://www.w3.org/TR/core-aam-1.2
  type: dfn
    text: computed role; url: /#roleMappingComputedRole
spec: MEDIAQUERIES4; urlPrefix: https://drafts.csswg.org/mediaqueries-4/
  type: dfn
    text: resolution media feature; url: #resolution
    text: media type; url: #media-type
    text: media feature; url: #media-features
    text: mf-name; url: #typedef-mf-name
spec: RFC9110; urlPrefix: https://httpwg.org/specs/rfc9110.html
  type: dfn
    text: field-name token; url: #fields.names
    text: method token; url: #method.overview
spec: SCREEN-CAPTURE; urlPrefix: https://www.w3.org/TR/screen-capture/
  type: dfn
    text: browser; url: #dfn-browser
    text: display surface; url: dfn-display-surface
spec: TOUCH-EVENTS; urlPrefix: https://www.w3.org/community/reports/touchevents/CG-FINAL-touch-events-20240704/
  type: dfn
    text: expose legacy touch event APIs; url: #conditionally-exposing-legacy-touch-event-apis
spec: WebSockets; urlPrefix: https://websockets.spec.whatwg.org/
  type: dfn
    text: WebSocket; url: #websocket
spec: WebTransport; urlPrefix: https://w3c.github.io/webtransport/
  type: dfn
    text: Cleanup WebTransport; url: #webtransport-cleanup
</pre>

<style>
var {
  color: #cd5c5c
}

/**
 * Emulate the appearace of the so-called "simple" table provided by ReSpec, as
 * used in WebDriver Classic.
 */
table.respec-simple {
  border-spacing: 0;
  border-collapse: collapse;
  border-bottom: 3px solid #005a9c;
}
table.respec-simple th {
  background: #005a9c;
  color: #fff;
  padding: 3px 5px;
  text-align: left;
}
table.respec-simple tr:nth-child(2n) {
  background: #f0f6ff;
}
table.respec-simple td {
  padding: 3px 10px;
  border-top: 1px solid #ddd;
}
</style>

# Introduction # {#intro}

<em>This section is non-normative.</em>

[[WEBDRIVER|WebDriver]] defines a protocol for introspection and
remote control of user agents. This specification extends WebDriver by
introducing bidirectional communication. In place of the strict
command/response format of WebDriver, this permits events to stream
from the user agent to the controlling software, better matching the
evented nature of the browser DOM.

# Infrastructure # {#infrastructure}

This specification depends on the Infra Standard. [[!INFRA]]

Network protocol messages are defined using CDDL. [[!RFC8610]]

This specification defines a <dfn>wait queue</dfn> which is a [=/map=].

Issue: Surely there's a better mechanism for doing this "wait for an event" thing.

<div algorithm>

When an algorithm |algorithm| running [=in parallel=] <dfn export>awaits</dfn> a set of
events |events|, and |resume id|:

1. Pause the execution of |algorithm|.

1. Assert: [=wait queue=] does not contain |resume id|.

1. Set [=wait queue=][|resume id|] to (|events|, |algorithm|).

</div>

<div algorithm>
To <dfn export>resume</dfn> given |name|, |id| and |parameters|:

1. If [=wait queue=] does not contain |id|, return.

1. Let (|events|, |algorithm|) be [=wait queue=][|id|]

1. For each |event| in |events|:

  1. If |event| equals |name|:

    1. Remove |id| from [=wait queue=].

    1. Resume running the steps in |algorithm| from the
       point at which they were paused, passing |name| and |parameters| as the
       result of the [=await=].

       Issue: Should we have something like microtasks to ensure this runs
       before any other tasks on the event loop?

</div>

A <dfn>WebDriver configuration</dfn> is a [=struct=] with:
* [=struct/item=] <dfn for="WebDriver configuration">global</dfn> which is a [=WebDriver configuration/value=], initially [=WebDriver configuration/unset=];
* [=struct/item=] <dfn for="WebDriver configuration">user contexts</dfn> which is a weak map between [=user context|user contexts=] and [=WebDriver configuration/value=], initially empty;
* [=struct/item=] <dfn for="WebDriver configuration">navigables</dfn> which is a weak map between [=/navigables=] and [=WebDriver configuration/value=], initially empty.

A [=WebDriver configuration=] has an <dfn for="WebDriver configuration">associated type</dfn> which
is a type.

The <dfn for="WebDriver configuration">value</dfn> for a [=WebDriver configuration=] is either a
value whose type is the [=WebDriver configuration/associated type=] for that configuration or
[=WebDriver configuration/unset=].

<dfn for="WebDriver configuration">Unset</dfn> is a value indicating that a specific configuration
[=WebDriver configuration/value=] has not been set.

<div algorithm>

Note: this algorithm allows accessing the [=WebDriver configuration=] for a given [=/navigable=]
by checking values in [=WebDriver configuration/navigables=], then in
[=WebDriver configuration/user contexts=] and finally in [=WebDriver configuration/global=].
Returns [=WebDriver configuration/unset=] if configuration is not set.

To <dfn>get WebDriver configuration value</dfn> of [=WebDriver configuration=] |configuration| for
[=/navigable=] |navigable|:

1. Let |top-level traversable| be |navigable|'s [=navigable/top-level traversable=].

1. If |configuration|'s [=WebDriver configuration/navigables=] [=map/contains=]
   |top-level traversable|:

   1. Let |navigable configuration value| be |configuration|'s
      [=WebDriver configuration/navigables=][|top-level traversable|].

   1. If |navigable configuration value| is not [=WebDriver configuration/unset=], return
      |navigable configuration value|.

1. Let |user context| be |navigable|'s [=associated user context=].

1. If |configuration|'s [=WebDriver configuration/user contexts=] [=map/contains=] |user context|:

   1. Let |user context configuration value| be |configuration|'s
      [=WebDriver configuration/user contexts=][|user context|].

   1. If |user context configuration value| is not [=WebDriver configuration/unset=], return
      |user context configuration value|.

1. Return |configuration|'s [=WebDriver configuration/global=].

</div>

<div algorithm>

Note: this is a generic algorithm for storing [=WebDriver configuration=] per |target|, which can be
either [=/navigable=], [=user context=], or store it globally if the |target| is null or omitted.

To <dfn for="WebDriver configuration">store</dfn> [=WebDriver configuration=] |configuration|'s
[=WebDriver configuration/value=] |value| in optional |target| which is a [=/navigable=],
a [=user context=] or null if not provided:

1. If |target| is null, set |configuration|'s [=WebDriver configuration/global=] to |value|.

1. If |target| is a [=user context=], set |configuration|'s
   [=WebDriver configuration/user contexts=][|target|] to |value|.

1. If |target| is a [=/navigable=], set |configuration|'s
   [=WebDriver configuration/navigables=][|target|] to |value|.

</div>

<div algorithm>

Note: This generic algorithm stores [=WebDriver configuration=]'s [=WebDriver configuration/value=]
in [=WebDriver configuration/global=], [=WebDriver configuration/user contexts=], or
[=WebDriver configuration/navigables=], depending on the presence of
"<code>userContexts</code>" and "<code>contexts</code>" in |command parameters|. These parameters
are mutually exclusive. If neither is provided, the configuration is stored globally.

To <dfn>store WebDriver configuration</dfn> [=WebDriver configuration=] |configuration|'s
[=WebDriver configuration/value=] |value| for given |command parameters|:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>" and |command parameters|
   [=map/contains=] "<code>contexts</code>", return [=error=] with [=error code=] [=invalid argument=].

1. Let |affected navigables| be an empty [=/set=].

1. If |command parameters| [=map/contains=] "<code>contexts</code>":

   1. Let |navigables| be the result of [=trying=] to [=get valid top-level traversables by ids=]
      with |command parameters|["<code>contexts</code>"].

   1. For each |navigable| of |navigables|:

      1. [=set/Append=] |navigable| to |affected navigables|.

      1. [=WebDriver configuration/Store=] |configuration|'s |value| in |navigable|.

1. Otherwise, if |command parameters| [=map/contains=] "<code>userContexts</code>":

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=] with
      |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. [=list/For each=] |top-level traversable| in the list of all [=/top-level traversables=] whose
         [=associated user context=] is |user context|:

         1. [=set/Append=] |top-level traversable| to |affected navigables|.

      1. [=WebDriver configuration/Store=] |configuration|'s |value| in |user context|.

1. Otherwise:

   1. [=list/For each=] |top-level traversable| of all [=/top-level traversables=], [=list/append=]
      |top-level traversable| to |affected navigables|.

   1. [=WebDriver configuration/Store=] |configuration|'s |value|.

1. Return |affected navigables|.

</div>

# Protocol # {#protocol}

This section defines the basic concepts of the WebDriver BiDi
protocol. These terms are distinct from their representation at the
<a href=#transport>transport</a> layer.

The protocol is defined using a [[!RFC8610|CDDL]] definition. For the
convenience of implementers two separate CDDL definitions are defined; the
<dfn cddl-module export lt="Remote end definition|remote end definition|remote-cddl">remote end definition</dfn> which defines the format of messages produced
on the [=local end=] and consumed on the [=remote end=], and the <dfn cddl-module export lt="Local end definition|local end definition|local-cddl">local end
definition</dfn> which defines the format of messages produced on the [=remote end=]
and consumed on the [=local end=]

## Definition ## {#protocol-definition}

Issue: Should this be an appendix?

This section gives the initial contents of the {^remote end definition^} and
{^local end definition^}. These are augmented by the definition fragments defined in
the remainder of the specification.

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
Command = {
  id: js-uint,
  CommandData,
  Extensible,
}

CommandData = (
  BrowserCommand //
  BrowsingContextCommand //
  EmulationCommand //
  InputCommand //
  NetworkCommand //
  ScriptCommand //
  SessionCommand //
  StorageCommand //
  WebExtensionCommand
)

EmptyParams = {
   Extensible
}
</pre>

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
Message = (
  CommandResponse /
  ErrorResponse /
  Event
)

CommandResponse = {
  type: "success",
  id: js-uint,
  result: ResultData,
  Extensible
}

ErrorResponse = {
  type: "error",
  id: js-uint / null,
  error: ErrorCode,
  message: text,
  ? stacktrace: text,
  Extensible
}

ResultData = (
  BrowserResult /
  BrowsingContextResult /
  EmulationResult /
  InputResult /
  NetworkResult /
  ScriptResult /
  SessionResult /
  StorageResult /
  WebExtensionResult
)

EmptyResult = {
  Extensible
}

Event = {
  type: "event",
  EventData,
  Extensible
}

EventData = (
  BrowsingContextEvent //
  InputEvent //
  LogEvent //
  NetworkEvent //
  ScriptEvent
)
</pre>

An <dfn export>EmptyResult</dfn> is a result type with no required fields,
used as the return type for commands that don't produce result data.

{^Remote end definition^} and {^Local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
Extensible = (*text => any)

js-int = -9007199254740991..9007199254740991
js-uint = 0..9007199254740991
</pre>

## Session ## {#session}

WebDriver BiDi extends the [=/session=] concept from [[WEBDRIVER|WebDriver]].

A [=/session=] has a <dfn>BiDi flag</dfn>, which is false unless otherwise
stated.

A <dfn export>BiDi session</dfn> is a [=/session=] which has the [=BiDi flag=]
set to true.

<div algorithm>
The list of <dfn export>active BiDi sessions</dfn> is given by:

1. Let |BiDi sessions| be a new [=/list=].

1. For each |session| in [=active sessions=]:

  1. If |session| is a [=BiDi session=] append |session| to |BiDi sessions|.

1. Return |BiDi sessions|.

</div>

## Modules ## {#protocol-modules}

The WebDriver BiDi protocol is organized into modules.

Each <dfn export>module</dfn> represents a collection of related
[=commands=] and [=events=] pertaining to a certain aspect of the user
agent. For example, a module might contain functionality for inspecting and
manipulating the DOM, or for script execution.

Each module has a <dfn for=module export>module name</dfn> which is a string. The
[=command name=] and [=event name=] for commands and events defined in the
module start with the [=module name=] followed by a period "<code>.</code>".

Modules which contain [=commands=] define {^remote end definition^}
fragments. These provide choices in the <code>CommandData</code> group for the
module's [=commands=], and can also define additional definition properties. They
can also define {^local end definition^} fragments that provide additional choices
in the <code>ResultData</code> group for the results of commands in the module.

Modules which contain events define {^local end definition^} fragments that are
choices in the <code>Event</code> group for the module's [=events=].

An implementation may define <dfn export>extension modules</dfn>. These must have a
[=module name=] that contains a single colon "<code>:</code>" character. The
part before the colon is the prefix; this is typically the same for all
[=extension modules=] specific to a given implementation and should be unique for a
given implementation.

Other specifications may define their own WebDriver-BiDi modules that extend the protocol.
Such modules must not have a name which contains a colon (<code>:</code>) character,
nor must they define [=command names=], [=event names=], or property names that contain that character.

Authors of external specifications are encouraged to to add new modules rather than extending
existing ones. Where it is desired to extend an existing module, it is preferred to integrate the
extension directly into the specification containing the original module definition.

## Commands ## {#commands}

A <dfn export>command</dfn> is an asynchronous operation, requested by
the [=local end=] and run on the [=remote end=], resulting in either a
result or an error being returned to the [=local end=]. Multiple
commands can run at the same time, and commands can potentially be
long-running. As a consequence, commands can finish out-of-order.

Each [=command=] is defined by:

- A <dfn export for=command>command type</dfn> which is defined by a {^remote end definition^}
   fragment containing a group. Each such group has two fields:
    - <code>method</code> which is a string literal of the form <code>[module
      name].[method name]</code>. This is the <dfn export for=command>command
      name</dfn>.
    - <code>params</code> which defines a mapping containing data that to be passed into
      the command. The populated value of this map is the
      <dfn export for=command>command parameters</dfn>.
- A <dfn export for=command>result type</dfn>, which is defined by a {^local
  end definition^} fragment.
- A set of [=remote end steps=] which define the actions to take for a command
  given a [=BiDi session=] and [=command parameters=] and return an
  instance of the command [=result type=].

A command that can run without an active session is a <dfn export>static
command</dfn>. Commands are not static commands unless stated in their
definition.

When commands are sent from the [=local end=] they have a command id. This is an
identifier used by the [=local end=] to identify the response from a particular
command. From the point of view of the [=remote end=] this identifier is opaque
and cannot be used internally to identify the command.

Note: This is because the command id is entirely controlled by the [=local end=]
and isn't necessarily unique over the course of a session. For example a [=local
end=] which ignores all responses could use the same command id for each command.

The <dfn export>set of all command names</dfn> is a [=/set=] containing
all the defined [=command names=], including any belonging to [=extension
modules=].

## Errors ## {#errors}

WebDriver BiDi extends the set of [=error codes=] from [[WEBDRIVER|WebDriver]]
with the following additional codes:

<dl>
  <dt><dfn for=errors export>invalid web extension</dfn>
  <dd>Tried to install an invalid web extension.

  <dt><dfn for=errors export>no such client window</dfn>
  <dd>Tried to interact with an unknown [=client window=].

  <dt><dfn for=errors export>no such handle</dfn>
  <dd>Tried to deserialize an unknown <code>RemoteObjectReference</code>.

  <dt><dfn for=errors export>no such history entry</dfn>
  <dd>Tried to havigate to an unknown [=session history entry=].

  <dt><dfn for=errors export>no such network collector</dfn>
  <dd>Tried to remove an unknown [=network/collector=].

  <dt><dfn for=errors export>no such intercept</dfn>
  <dd>Tried to remove an unknown [=network intercept=].

  <dt><dfn for=errors export>no such network data</dfn>
  <dd>Tried to reference an unknown [=network data=].

  <dt><dfn for=errors export>no such node</dfn>
  <dd>Tried to deserialize an unknown <code>SharedReference</code>.

  <dt><dfn for=errors export>no such request</dfn>
  <dd>Tried to continue an unknown [=/request=].

  <dt><dfn for=errors export>no such screencast</dfn>
  <dd>Tried to stop an unknown screencast recording.

  <dt><dfn for=errors export>no such script</dfn>
  <dd>Tried to remove an unknown [=preload script=].

  <dt><dfn for=errors export>no such storage partition</dfn>
  <dd>Tried to access data in a non-existent storage partition.

  <dt><dfn for=errors export>no such user context</dfn>
  <dd>Tried to reference an unknown [=user context=].

  <dt><dfn for=errors export>no such web extension</dfn>
  <dd>Tried to reference an unknown web extension.

  <dt><dfn for=errors export>unable to close browser</dfn>
  <dd>Tried to close the browser, but failed to do so.

  <dt><dfn for=errors export>unable to set cookie</dfn>
  <dd>Tried to create a cookie, but the user agent rejected it.

  <dt><dfn for=errors export>underspecified storage partition</dfn>
  <dd>Tried to interact with data in a storage partition which was not adequately specified.

  <dt><dfn for=errors export>unable to set file input</dfn>
  <dd>Tried to set a file input, but failed to do so.

  <dt><dfn for=errors export>unavailable network data</dfn>
  <dd>Tried to get network data which was not collected or already evicted.

</dl>

<pre class="cddl" data-cddl-module="local-cddl">
ErrorCode = "invalid argument" /
            "invalid selector" /
            "invalid session id" /
            "invalid web extension" /
            "move target out of bounds" /
            "no such alert" /
            "no such network collector" /
            "no such element" /
            "no such frame" /
            "no such handle" /
            "no such history entry" /
            "no such intercept" /
            "no such network data" /
            "no such node" /
            "no such request" /
            "no such screencast" /
            "no such script" /
            "no such storage partition" /
            "no such user context" /
            "no such web extension" /
            "session not created" /
            "unable to capture screen" /
            "unable to close browser" /
            "unable to set cookie" /
            "unable to set file input" /
            "unavailable network data" /
            "underspecified storage partition" /
            "unknown command" /
            "unknown error" /
            "unsupported operation"
</pre>

## Events ## {#events}

An <dfn export>event</dfn> is a notification, sent by the [=remote end=]
to the [=local end=], signaling that something of interest has
occurred on the [=remote end=].

 - An <dfn export for=event>event type</dfn> is defined by a {^local
   end definition^} fragment containing a group. Each such group has two fields:
    - <code>method</code> which is a string literal of the form <code>[module
      name].[event name]</code>. This is the <dfn export for=event>event
      name</dfn>.

    - <code>params</code> which defines a mapping containing event data. The
      populated value of this map is the <dfn export for=event>event
      parameters</dfn>.
 - A <dfn for=event export>remote end event trigger</dfn> which defines when the
   event is triggered and steps to construct the [=event type=] data.
 - Optionally, a set of <dfn for=event export>remote end subscribe steps</dfn>,
   which define steps to take when a local end subscribes to an event. Where
   defined these steps have an associated <dfn for=event export>subscribe
   priority</dfn> which is an integer controlling the order in which the steps
   are run when multiple events are enabled at once, with lower integers
   indicating steps that run earlier.

A [=BiDi session=] has <dfn for=event>subscriptions</dfn> which is a [=/list=] of [=subscriptions=].

A [=BiDi session=] has a <dfn for=event>known subscription ids</dfn> which is a [=/set=] of
all [=subscription/subscription ids=] that have been issued to the [=local end=] but which have not yet been unsubscribed.

A <dfn for=event>subscription</dfn> is a [=/struct=] consisting of a
<dfn for=subscription>subscription id</dfn> (a string),
<dfn for=subscription>event names</dfn> (a [=/set=] of event names),
<dfn for=subscription>top-level traversable ids</dfn> (a [=/set=] of IDs of [=/top-level traversables=])
and <dfn for=subscription>user context ids</dfn> (a [=/set=] of IDs of [=user context|user contexts=]).

A [=subscription=] |subscription| is <dfn for="subscription">global</dfn>
if |subscription|'s [=subscription/top-level traversable ids=] is an empty set
and |subscription|'s [=user context ids=] is an empty set.

<div algorithm>

The <dfn export>set of sessions for which an event is enabled</dfn> given |event name| and |navigables| is:

1. Let |sessions| be a new [=/set=].

1. For each |session| in [=active BiDi sessions=]:

  1. If [=event is enabled=] with |session|, |event name| and |navigables|,
    append |session| to |sessions|.

1. Return |sessions|.

</div>

<div algorithm>

To determine if an <dfn export>event is enabled</dfn> given |session|,
|event name| and |navigables|:

Note: |navigables| is a set because a [=shared worker=] can be associated
      with multiple contexts.

1. Let |top-level traversables| be [=get top-level traversables=] with |navigables|.

1. For each |subscription| in |session|'s [=subscriptions=]:

   1. If |subscription|'s [=subscription/event names=] do not [=set/contains=] |event name|, [=continue=].

   1. If |subscription| is [=subscription/global=] return true.

   1. If [=subscription/user context ids=] is not empty:

      1. [=list/For each=] |navigable| in |top-level traversables|:

         1. If |subscription|'s [=subscription/user context ids=] [=set/contains=]
            |navigable|'s [=associated user context=]'s [=user context id=], return true.

   1. Otherwise:

      1. Let |subscription top-level traversables| be [=get navigables by ids=] with |subscription|'s [=subscription/top-level traversable ids=].

      1. If the [=set/intersection=] of |top-level traversables| and
         |subscription top-level traversables| is not [=list/empty=] return true.

1. Return false.

</div>

<div algorithm>

The <dfn export>set of top-level traversables for which an event is enabled</dfn>
given |event name| and |session| is:

1. Let |result| be a new [=/set=].

1. For each |subscription| in |session|'s [=subscriptions=]:

   1. If |subscription|'s [=subscription/event names=] [=set/contains|does not contain=] |event name|,
      [=continue=].

   1. If |subscription|'s is [=subscription/global=]:

      1. For each |traversable| in remote end's [=/top-level traversables=]:

        1. [=set/Append=] |traversable| to |result|.

      1. [=Break=].

   1. Otherwise, if [=subscription/user context ids=] is not empty:

      1. For each |traversable| in remote end's [=/top-level traversables=]:

         1. [=set/Append=] |traversable| to |result| if
            |subscription|'s [=subscription/user context ids=] [=set/contains=]
            |traversable|'s [=associated user context=]'s [=user context id=].

   1. Otherwise:

      1. Let |top-level traversables| be [=get navigables by ids=] with |subscription|'s [=subscription/top-level traversable ids=].

      1. [=set/Append=] each item of |top-level traversables| to |result|.

1. Return |result|.

</div>

<div algorithm>
To <dfn>obtain a set of event names</dfn> given a |name|:

1. Let |events| be an empty [=/set=].

1. If |name| contains a U+002E (period):

  1. If |name| is the [=event name=] for an event, append |name| to |events|
     and return [=success=] with data |events|.

  1. Return an [=error=] with [=error code=] [=invalid argument=]

1. Otherwise |name| is interpreted as representing all the events in a
   module. If |name| is not a [=module name=] return an [=error=] with
   [=error code=] [=invalid argument=].

1. Append the [=event name=] for each [=event=] in the module with name |name| to
   |events|.

1. Return [=success=] with data |events|.

</div>

# Transport # {#transport}

Message transport is provided using the WebSocket protocol.
[[!RFC6455]]

Note: In the terms of the WebSocket protocol, the [=local end=] is the
client and the [=remote end=] is the server / remote host.

Note: The encoding of [=commands=] and [=events=] as messages is
similar to JSON-RPC, but this specification does not normatively
reference it. [[JSON-RPC]] The normative requirements on [=remote ends=]
are instead given as a precise processing model, while no
normative requirements are given for [=local ends=].

A <dfn>WebSocket listener</dfn> is a network endpoint that is able
to accept incoming [[!RFC6455|WebSocket]] connections.

A [=WebSocket listener=] has a <dfn for=listener>host</dfn>, a <dfn
for=listener>port</dfn>, a <dfn for=listener>secure flag</dfn>, and a
<dfn>list of WebSocket resources</dfn>.

When a [=WebSocket listener=] |listener| is created, a [=remote end=]
must start to listen for WebSocket connections on the host and port
given by |listener|'s [=listener/host=] and [=listener/port=]. If
|listener|'s [=listener/secure flag=] is set, then connections
established from |listener| must be TLS encrypted.

A [=remote end=] has a [=/set=] of [=WebSocket listeners=] <dfn>active
listeners</dfn>, which is initially empty.

A [=remote end=] has a [=/set=] of <dfn>WebSocket connections not associated with a
session</dfn>, which is initially empty.

A <dfn>WebSocket connection</dfn> is a network connection that follows the
requirements of the [[!RFC6455|WebSocket protocol]]

A [=BiDi session=] has a [=/set=] of <dfn>session WebSocket
connections</dfn> whose elements are [=WebSocket connections=]. This is
initially empty.

A [=BiDi session=] |session| is <dfn>associated with connection</dfn>
|connection| if |session|'s [=session WebSocket connections=] contains |connection|.

Note: Each [=WebSocket connection=] is associated with at most one [=BiDi
session=].

<div>

When a client [=establishes a WebSocket connection=] |connection| by
connecting to one of the set of [=active listeners=] |listener|, the
implementation must proceed according to the WebSocket [=server-side
requirements=], with the following steps run when deciding whether to
accept the incoming connection:

1. Let |resource name| be the resource name from [=reading the
   client's opening handshake=]. If |resource name| is not in
   |listener|'s [=list of WebSocket resources=], then stop
   running these steps and act as if the requested service is not
   available.

1. If |resource name| is the byte string "<code>/session</code>",
   and the implementation [=supports BiDi-only sessions=]:

    1. Run any other implementation-defined steps to decide if the
       connection should be accepted, and if it is not stop running these
       steps and act as if the requested service is not available.

    1. Add the connection to [=WebSocket connections not associated with a
       session=].

    1. Return.

1. [=Get a session ID for a WebSocket resource=] with |resource name|
   and let |session id| be that value. If |session id| is null then
   stop running these steps and act as if the requested service is not
   available.

1. If there is a [=/session=] in the list of [=active sessions=] with
   |session id| as its [=session ID=] then let |session| be that
   session. Otherwise stop running these steps and act as if the
   requested service is not available.

1. Run any other implementation-defined steps to decide if the
   connection should be accepted, and if it is not stop running these
   steps and act as if the requested service is not available.

1. Otherwise append |connection| to |session|'s [=session WebSocket
   connections=], and proceed with the WebSocket [=server-side requirements=]
   when a server chooses to accept an incoming connection.

Issue: Do we support > 1 connection for a single session?

</div>

When [=a WebSocket message has been received=] for a [=WebSocket
connection=] |connection| with type |type| and data |data|, a [=remote end=]
must [=handle an incoming message=] given |connection|, |type| and |data|.

When [=the WebSocket closing handshake is started=] or when [=the
WebSocket connection is closed=] for a [=WebSocket connection=]
|connection|, a [=remote end=] must [=handle a connection closing=]
given |connection|.

Note: Both conditions are needed because it is possible for a
WebSocket connection to be closed without a closing handshake.

<div algorithm>

To <dfn>construct a WebSocket resource name</dfn>
given a [=/session=] |session|:

1. If |session| is null, return "<code>/session</code>"

1. Return the result of concatenating the string "<code>/session/</code>"
   with |session|'s [=session ID=].

</div>

<div algorithm>

To <dfn>construct a WebSocket URL</dfn> given a [=WebSocket listener=]
|listener| and [=/session=] |session|:

1. Let |resource name| be the result of [=construct a WebSocket
   resource name=] with |session|.

1. Return a [=WebSocket URI=] constructed with host set to
   |listener|'s [=listener/host=], port set to |listener|'s
   [=listener/port=], path set to |resource name|, following the wss-URI
   construct if |listener|'s [=listener/secure flag=] is set and the ws-URL
   construct otherwise.

</div>

<div algorithm>

To <dfn>get a session ID for a WebSocket resource</dfn>
given |resource name|:

1. If |resource name| doesn't begin with the byte string
   "<code>/session/</code>", return null.

1. Let |session id| be the bytes in |resource name| following the
   "<code>/session/</code>" prefix.

1. If |session id| is not the string representation of a
   [[!RFC9562|UUID]], return null.

1. Return |session id|.

</div>

<div algorithm>
To <dfn>start listening for a WebSocket connection</dfn> given a
[=/session=] |session|:

1. If there is an existing [=WebSocket listener=] in [=active listeners=] which
   the [=remote end=] would like to reuse, let |listener| be that
   listener. Otherwise let |listener| be a new [=WebSocket listener=] with
   [=implementation-defined=] [=listener/host=], [=listener/port=],
   [=listener/secure flag=], and an empty [=list of WebSocket resources=].

1. Let |resource name| be the result of [=construct a WebSocket
   resource name=] with |session|.

1. Append |resource name| to the [=list of WebSocket resources=] for
   |listener|.

1. [=set/Append=] |listener| to the [=remote end=]'s [=active
    listeners=].

1. Return |listener|.

</div>

Note: An [=intermediary node=] handling multiple sessions can use one
or many WebSocket listeners. [[!WEBDRIVER|WebDriver]] defines that
an [=endpoint node=] supports at most one session at a time, so it's
expected to only have a single listener.

Note: For an [=endpoint node=] the [=listener/host=] in the above steps will
typically be "<code>localhost</code>".

<div algorithm>
To <dfn>handle an incoming message</dfn> given a [=WebSocket connection=]
|connection|, type |type| and data |data|:

1. If |type| is not [=%x1 denotes a text frame|text=], [=send an error
   response=] given |connection|, null, and [=invalid argument=], and finally
   return.

1. [=Assert=]: |data| is a [=scalar value string=], because the
    WebSocket [=handling errors in UTF-8-encoded data=] would already
    have [=fail the WebSocket connection|failed the WebSocket
    connection=] otherwise.

   Issue: Nothing seems to define what [=status codes|status code=]
   is used for UTF-8 errors.

1. If there is a [=BiDi Session=] [=associated with connection=] |connection|,
   let |session| be that session. Otherwise if |connection| is in [=WebSocket
   connections not associated with a session=], let |session| be
   null. Otherwise, return.

1. Let |parsed| be the result of [=parse JSON into Infra values|parsing JSON
   into Infra values=] given |data|. If this throws an exception, then [=send
   an error response=] given |connection|, null, and [=invalid argument=], and
   finally return.

1. If |session| is not null and not in [=active sessions=] then return.

1. Match |parsed| against the {^remote end definition^}. If this results in a
   match:

   1. Let |matched| be the [=/map=] representing the matched data.

   1. Assert: |matched| [=map/contains=] "<code>id</code>", "<code>method</code>", and
      "<code>params</code>".

   1. Let |command id| be |matched|["<code>id</code>"].

   1. Let |method| be |matched|["<code>method</code>"]

   1. Let |command| be the command with [=command name=] |method|.

   1. If |session| is null and |command| is not a [=static command=], then
      [=send an error response=] given |connection|, |command id|, and [=invalid
      session id=], and return.

   1. Run the following steps in parallel:

     1. Let |result| be the result of running the [=remote end steps=] for
        |command| given |session| and [=command parameters=]
        |matched|["<code>params</code>"]

    1. If |result| is an [=error=], then [=send an error response=] given
       |connection|, |command id|, and |result|'s [=error code=], and finally
       return.

    1. Let |value| be |result|'s data.

    1. Assert: |value| matches the definition for the [=result type=]
       corresponding to the command with [=command name=] |method|.

    1. If |method| is "<code>session.new</code>", let |session| be the entry in
       the list of [=active sessions=] whose [=session ID=] is equal to the
       "<code>sessionId</code>" property of |value|, [=set/append=]
       |connection| to |session|'s [=session WebSocket connections=], and
       remove |connection| from the [=WebSocket connections not associated with
       a session=].

    1. Let |response| be a new [=/map=] matching the <code>CommandResponse</code>
       production in the {^local end definition^} with the <code>id</code>
       field set to |command id| and the <code>value</code> field set to
       |value|.

    1. Let |serialized| be the result of [=serialize an infra value to JSON
       bytes=] given |response|.

    1. [=Send a WebSocket message=] comprised of |serialized| over
       |connection|.

1. Otherwise:

   1. Let |command id| be null.

   1. If |parsed| is a [=/map=] and |parsed|["<code>id</code>"] exists and is an
      integer greater than or equal to zero, set |command id| to that integer.

   1. Let |error code| be [=invalid argument=].

   1. If |parsed| is a [=/map=] and |parsed|["<code>method</code>"] exists and is a
      string, but |parsed|["<code>method</code>"] is not in the [=set of all
      command names=], set |error code| to [=unknown command=].

   1. [=Send an error response=] given |connection|, |command id|, and
      |error code|.

</div>

<div algorithm>

To <dfn>get related navigables</dfn> given an [=script/settings object=]
|settings|:

1. Let |related navigables| be an empty [=/set=].

1. If |settings|' [=relevant global object=] is a {{Window}}:

   1. Let |navigable| be [=relevant global object=]'s <a>associated
      <code>Document</code></a>'s [=/node navigable=].

   1. If |navigable| is not null, append |navigable| to |related navigables|.

1. Otherwise if the [=realm/global object=] specified by |settings| is a
   {{WorkerGlobalScope}}, for each |owner| in the [=realm/global object=]'s
   [=owner set=]:

   1. Let |navigable| be null.

   1. If |owner| is a [=/Document=], set |navigable| to |owner|'s
      [=/node navigable=].

   1. If |navigable| is not null, append |navigable| to |related navigables|.


1. Return |related navigables|.

</div>

<div algorithm>

To <dfn>get navigables by ids</dfn> given a [=/list=] of context ids |navigable ids|:

1. Let |result| be an empty [=/set=].

1. For each |navigable id| in |navigable ids|:

   1. Let |navigable| be the [=/navigable=] with id |navigable id|
      if such [=/navigable=] exists, and null otherwise.

   1. [=set/Append=] |navigable| to |result| if |navigable| is not null.

1. Return |result|.

</div>

<div algorithm>

To <dfn>get top-level traversables</dfn> given a [=/list=] of [=/navigables=] |navigables|:

1. Let |result| be an empty [=/set=].

1. For each |navigable| in |navigables|:

   1. [=set/Append=] |navigable|'s [=navigable/top-level traversable=] to |result|.

1. Return |result|.

</div>

<div algorithm>

To <dfn>get valid navigables by ids</dfn> given a [=/list=] of context ids |navigable ids|:

1. Let |result| be an empty [=/set=].

1. For each |navigable id| in |navigable ids|:

   1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with |navigable id|.

   1. [=set/Append=] |navigable| to |result|.

1. Return [=success=] with data |result|.

</div>

<div algorithm>

To <dfn>get valid top-level traversables by ids</dfn> given a [=/list=] of context ids |navigable ids|:

1. Let |result| be an empty [=/set=].

1. For each |navigable id| in |navigable ids|:

   1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with |navigable id|.

   1. If |navigable| is not a [=/top-level traversable=], return [=error=]
      with [=error code=] [=invalid argument=].

   1. [=set/Append=] |navigable| to |result|.

1. Return [=success=] with data |result|.

</div>

<div algorithm> To <dfn export>emit an event</dfn> given |session|, and |body|:

1. [=Assert=]: |body| matches the <code>Event</code> production.

1. Let |serialized| be the result of [=serialize an infra value to JSON
   bytes=] given |body|.

1. [=list/For each=] |connection| in |session|'s [=session WebSocket connections=]:

  1. [=Send a WebSocket message=] comprised of |serialized| over |connection|.

</div>

<div algorithm>
To <dfn>send an error response</dfn> given a [=WebSocket connection=]
|connection|, |command id|, and |error code|:

1. Let |error data| be a new [=/map=] matching the <code>ErrorResponse</code>
   production in the {^local end definition^}, with the <code>id</code> field
   set to |command id|, the <code>error</code> field set to |error code|, the
   <code>message</code> field set to an implementation-defined string
   containing a human-readable definition of the error that occurred and the
   <code>stacktrace</code> field optionally set to an implementation-defined
   string containing a stack trace report of the active stack frames at the
   time when the error occurred.

1. Let |response| be the result of [=serialize an infra value to JSON bytes=]
   given |error data|.

   Note: |command id| can be null, in which case the <code>id</code> field will
   also be set to null, not omitted from |response|.

1. [=Send a WebSocket message=] comprised of |response| over |connection|.

</div>


<div algorithm>

To <dfn>handle a connection closing</dfn> given a [=WebSocket connection=]
|connection|:

1. If there is a [=BiDi session=] [=associated with connection=] |connection|:

  1. Let |session| be the [=BiDi session=] [=associated with connection=]
     |connection|.

  1. Remove |connection| from |session|'s [=session WebSocket
     connections=].

1. Otherwise, if [=WebSocket connections not associated with a session=]
   [=list/contains=] |connection|, [=list/remove=] |connection| from that set.

Note: This does not end any [=/session=].

Issue: Need to hook in to the session ending to allow the UA to close
the listener if it wants.

</div>

<div algorithm>

To <dfn>close the WebSocket connections</dfn> given |session|:

1. For each |connection| in |session|'s [=session WebSocket
   connections=]:

  1. [=Start the WebSocket closing handshake=] with |connection|.

     Note: this will result in the steps in [=handle a connection closing=]
     being run for |connection|, which will clean up resources associated with
     |connection|.

</div>

## Establishing a Connection ## {#establishing}

WebDriver clients opt in to a bidirectional connection by requesting the
[=WebSocket URL=] capability with value true.

<div algorithm="webSocketUrl new session algorithm">
The [=WebDriver new session algorithm=] defined by this specification,
with parameters |session|, |capabilities|, and |flags| is:

1. If |flags| contains "<code>bidi</code>", return.

1. Let |webSocketUrl| be the result of [=getting a property=] named
   "<code>webSocketUrl</code>" from |capabilities|.

1. If |webSocketUrl| is undefined, return.

1. [=Assert=]: |webSocketUrl| is true.

1. Let |listener| be the result of [=start listening for a WebSocket
   connection=] given |session|.

1. Set |webSocketUrl| to the result of [=construct a WebSocket
   URL=] with |listener| and |session|.

1. [=Set a property=] on |capabilities| named
   "<code>webSocketUrl</code>" to |webSocketUrl|.

1. Set |session|'s [=BiDi flag=] to true.

1. Append "<code>bidi</code>" to flags.

</div>

<div algorithm="no HTTP new session">

Implementations should also allow clients to establish a [=BiDi Session=] which
is not a [=HTTP Session=]. In this case the URL to the WebSocket server is
communicated out-of-band. An implementation that allows this <dfn>supports
BiDi-only sessions</dfn>. At the time such an implementation is ready to accept
requests to start a WebDriver session, it must:

1. [=Start listening for a WebSocket connection=] given null.

</div>

# Sandboxed Script Execution # {#sandbox}

A common requirement for automation tools is to execute scripts which have
access to the DOM of a document, but don't have information about any changes to
the DOM APIs made by scripts running in the navigable containing the
document.

A [=BiDi session=] has a <dfn>sandbox map</dfn> which is a weak map in which the
keys are {{Window}} objects, and the values are maps between strings and
{{SandboxWindowProxy}} objects.

Note: The definition of sandboxes here is an attempt to codify the behaviour of
existing implementations. It exposes parts of the implementations that have
previously been considered internal by specifications, in particular the
distinction between the internal state of platform objects (which is typically
implemented as native objects in the main implementation language of the browser
engine) and the ECMAScript-visible state. Because existing sandbox
implementations happen at a low level in the engine, implementations converging
toward the specification in all details might be a slow process. In the
meantime, implementers are encouraged to provide detailed documentation on any
differences with the specification, and users of this feature are encouraged to
explicitly test that scripts running in sandboxes work in all implementations.

## Sandbox Realms ## {#sandbox-realm}

Each sandbox is a unique ECMAScript [=Realm=]. However the sandbox realm
provides access to platform objects in an existing {{Window}} realm via
{{SandboxProxy}} objects.

<div algorithm>
To <dfn>get or create a sandbox realm</dfn> given |name| and |navigable|:

1. If |name| is an empty string, then return [=error=] with
   [=error code=] [=invalid argument=].

1. Let |window| be |navigable|'s [=active window=].

1. If [=sandbox map=] does not contain |window|, set [=sandbox map=][|window|]
   to a new [=/map=].

1. Let |sandboxes| be [=sandbox map=][|window|].

1. If |sandboxes| does not contain |name|, set |sandboxes|[|name|] to [=create
   a sandbox realm=] with |navigable|.

1. Return [=success=] with data |sandboxes|[|name|].

</div>

<div algorithm>
To <dfn>create a sandbox realm</dfn> with |window|:

Issue: Define creation of sandbox realm. This is going to return a
{{SandboxWindowProxy}} wrapping |window|.

</div>

<div algorithm>

To <dfn>get a sandbox name</dfn> given |target realm|:

1. Let |realms maps| be [=get the values=] of [=sandbox map=].

1. For each |realms map| in |realms maps|:

  1. For each |name| → |realm| in |realms map|:

    1. If |realm| is |target realm|, return |name|.

1. Return null.

</div>

## Sandbox Proxy Objects ## {#sandbox-proxy}

A <dfn interface>SandboxProxy</dfn> object is an exotic object that mediates sandboxed access to
objects from another realm. Sandbox proxy objects are designed to enforce the
following restrictions:

* Platform objects are accessible, but property access returns only
  Web IDL-defined properties and not ECMAScript-defined properties (either
  "expando" properties that are not present in the underlying interface, or
  ECMAScript-defined properties that shadow a property in the underlying
  interface).

* Setting a property either runs Web IDL-defined setter steps, or sets a property
  on the proxy object. This means that properties written outside the sandbox
  are not accessible, but interface members can be used as normal.

There is no {{SandboxProxy}} interface object.

Issue: Define in detail how {{SandboxProxy}} works

<div algorithm>
To get <dfn>unwrapped</dfn> |object|:

1. While |object| is {{SandboxProxy}} or {{SandboxWindowProxy}}, set |object| to it's
   wrapped object.

1. Return |object|.

</div>


## SandboxWindowProxy ## {#sandbox-sandboxwindowproxy}

A <dfn interface>SandboxWindowProxy</dfn> is an exotic object that represents a
{{Window}} object wrapped by a {{SandboxProxy}} object. This provides sandboxed
access to that data in a {{Window}} global.

Issue: Define how this works.

# User Contexts # {#user-contexts}

A <dfn export>user context</dfn> represents a collection of zero or more
[=/top-level traversables=] within a [=remote end=]. Each [=user context=] has
an associated [=storage partition=], so that [=remote end=] data is not shared
between different [=user context|user contexts=].

Issue: Unclear that this is the best way to formally define the concept of a
user context or the interaction with storage.

Note: The infra spec uses the term "user agent" to refer to the same concept as
[=user context|user contexts=]. However, this is not compatible with usage of
the term "user agent" to mean the entire web client with multiple [=user
context|user contexts=]. Although this difference is not visible to web content,
it is observed via WebDriver, so we avoid using this terminology.

A [=user context=] has a <dfn export for="user context">user context id</dfn>,
which is a unique string set upon the user context creation.

A [=/navigable=] has an <dfn export>associated user context</dfn>, which is a [=user
context=].

When a new [=/top-level traversable=] is created its [=associated user context=]
is set to a user context in the [=set of user contexts=].

Note: In some cases the user context is set by specification when the
[=/top-level traversable=] is created, however in cases where no such
requirements are present, the [=associated user context=] for a [=/top-level
traversable=] is implemenation-defined.

Issue: Should we specify that [=/top-level traversables=] with a non-null
opener have the same [=associated user context=] as their opener?
Need to check if this is something existing implementations enforce.

A [=child navigable=]'s [=associated user context=] is it's
[=navigable/parent=]'s [=associated user context=].

A [=user context=] which isn't the [=associated user context=] for any
[=/top-level traversable=] is an <dfn>empty user context</dfn>.

The <dfn>default user context</dfn> is a [=user context=] with [=user context
id=] <code>"default"</code>.

An implementation has a <dfn>set of user contexts</dfn>, which is a [=/set=] of
[=user context|user contexts=]. Initially this contains the [=default user context=].

Implementations may [=set/append=] new [=user context|user contexts=] to the [=set of user
contexts=] at any time, for example in response to user actions.

Note: "At any time" here includes during implementation startup, so a given
implementation might always have multiple entries in the [=set of user contexts=].

Implementations may [=set/remove=] any [=empty user context=], with exception of
the [=default user context=], from the [=set of user contexts=] at any
time. However they are not required to remove such [=user context|user contexts=]. [=user context|User contexts=] that are not [=empty user contexts=] must not be removed from the
[=set of user contexts=].

A [=BiDi session=] has a
<dfn>user context to accept insecure certificates override map</dfn>, which is a
[=/map=] between [=user context|user contexts=] and boolean.

A [=BiDi session=] has a <dfn>user context to proxy configuration map</dfn>, which is
a [=/map=] between [=user context|user contexts=] and [=proxy configuration=].

An <dfn>emulated network conditions struct</dfn> is a [=struct=] with:
* [=struct/item=] named <dfn id="emulated-network-conditions-struct-offline"
  for="emulated-network-conditions-struct">offline</dfn> which is a boolean or null.

A [=BiDi session=] has a <dfn for=session>emulated network conditions</dfn> which is
a [=struct=] with an [=struct/item=] named <dfn
for="emulated network conditions">default network conditions</dfn>, which is an
[=emulated network conditions struct=] or null, an [=struct/item=] named
<dfn for="emulated network conditions">user context network conditions</dfn>, which
is a weak map between [=user context|user contexts=] and [=emulated network conditions struct=],
and a [=struct/item=] named
<dfn for="emulated network conditions">navigable network conditions</dfn>, which is a
weak map between [=/navigables=] and [=emulated network conditions struct=].

When a [=user context=] is [=set/remove|removed=] from the
[=set of user contexts=], [=remove user context subscriptions=].

<div algorithm>

To <dfn export>remove user context subscriptions</dfn>:

1. For each |session| in [=active sessions=]:

  1. Let |subscriptions to remove| be a [=/set=].

  1. For each |subscription| in |session|'s [=subscriptions=]:

     1. If |subscription|'s [=subscription/user context ids=] [=set/contains=] |navigable|'s [=associated user context=]'s [=user context id=];

        1. [=set/Remove=] |navigable|'s [=associated user context=]'s [=user context id=] from |subscription|'s [=subscription/user context ids=].

        1. If |subscription|'s [=subscription/user context ids=] is empty:

           1. [=set/Append=] |subscription| to |subscriptions to remove|.

  1. [=list/Remove=] |subscriptions to remove| from |session|'s [=subscriptions=].

</div>

<div algorithm>
To <dfn export>get user context</dfn> given |user context id|:

1. For each |user context| in the [=set of user contexts=]:

 1. If |user context|'s [=user context id=] equals |user context id|:

   1. Return |user context|.

1. Return null.

</div>

<div algorithm>

To <dfn export>get valid user contexts</dfn> given |user context ids|:

1. Let |result| be an empty [=/set=].

1. For each |user context id| of |user context ids|:

   1. Set |user context| to [=get user context=] with |user context id|.

   1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

   1. [=set/Append=] |user context| to |result|.

1. Return |result|.

</div>

# Modules # {#modules}

## The session Module ## {#module-session}

The <dfn export for=modules>session</dfn> module contains commands and
events for monitoring the status of the remote end.

### Definition ### {#module-session-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
SessionCommand = (
  session.End //
  session.New //
  session.Status //
  session.Subscribe //
  session.Unsubscribe
)
</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
SessionResult = (
  session.EndResult /
  session.NewResult /
  session.StatusResult /
  session.SubscribeResult /
  session.UnsubscribeResult
)
</pre>

<div algorithm>
To <dfn>end the session</dfn> given |session|:

1. Remove |session| from [=active sessions=].

1. If [=active sessions=] is [=list/empty=], set the [=webdriver-active flag=]
   to false.

</div>

<div algorithm>

To <dfn>cleanup the session</dfn> given |session|:

1. [=Close the WebSocket connections=] with |session|.

1. For each |user context| in the [=set of user contexts=]:

   1. [=map/Remove=] |session|'s
      [=user context to accept insecure certificates override map=][|user context|].

   1. [=map/Remove=] |session|'s
      [=user context to proxy configuration map=][|user context|].

1. For each |request id| → (<var ignore>request</var>, <var ignore>phase</var>, |response|)
   in |session|'s [=blocked request map=]:

   1. [=Resume=] with "<code>continue request</code>", |request id|
      and (|response|, "<code>incomplete</code>").

1. For each |collector| in |session|'s [=network collectors=]:

   1. Let |collector id| be |collector|'s [=network-collector/collector=].

   1. For each |collected data| in [=collected network data=], [=remove collector from data=]
      with |collected data| and |collector id|.

1. For each |screencast recording| in |session|'s [=screencast recordings map=]:

   1. [=Stop a screencast recording=] given |screencast recording|.

   1. [=map/Remove=] |screencast recording| from [=screencast recordings map=].

1. If [=active sessions=] is [=list/empty=], [=cleanup remote end state=].

1. Perform any implementation-specific cleanup steps.

</div>

<div algorithm>
To <dfn>cleanup remote end state</dfn>.

1. [=map/Clear=] the [=before request sent map=].

1. Set the [=default cache behavior=] to "<code>default</code>".

1. [=map/Clear=] the [=navigable cache behavior map=].

1. Perform implementation-defined steps to enable any
   implementation-specific resource caches that are usually enabled in the
   current [=remote end=] configuration.

</div>

### Types ### {#module-session-types}

#### The session.CapabilitiesRequest Type #### {#type-session-CapabilitiesRequest}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.CapabilitiesRequest = {
  ? alwaysMatch: session.CapabilityRequest,
  ? firstMatch: [*session.CapabilityRequest]
}
</pre>

The <code>session.CapabilitiesRequest</code> type represents the capabilities requested
for a session.

#### The session.CapabilityRequest Type #### {#type-session-CapabilityRequest}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.CapabilityRequest = {
  ? acceptInsecureCerts: bool,
  ? browserName: text,
  ? browserVersion: text,
  ? platformName: text,
  ? proxy: session.ProxyConfiguration,
  ? unhandledPromptBehavior: session.UserPromptHandler,
  Extensible
}
</pre>

The <code>session.CapabilityRequest</code> type represents a specific set of
requested capabilities.

WebDriver BiDi defines [=additional WebDriver capability|additional WebDriver
capabilities=]. The following tables enumerates the capabilities each
implementation must support for WebDriver BiDi.

<pre class=simpledef>
Capability: <dfn>WebSocket URL</dfn>
Key: "<code>webSocketUrl</code>"
Value type: boolean
Description: Defines the current session's support for bidirectional connection.
</pre>

<div algorithm="webSocketUrl capability deserialization algorithm">
The [=additional capability deserialization algorithm=] for the
"<code>webSocketUrl</code>" capability, with parameter |value| is:

1. If |value| is not a boolean, return [=error=] with [=error code|code=]
   [=invalid argument=].

1. Return [=success=] with data |value|.

</div>

<div algorithm="webSocketUrl capability serialization algorithm">
The [=matched capability serialization algorithm=] for the "<code>webSocketUrl</code>" capability,
with parameter |value| is:

1. If |value| is false, return [=success=] with data null.

1. Return [=success=] with data true.

</div>

#### The session.ProxyConfiguration Type #### {#type-session-ProxyConfiguration}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.ProxyConfiguration = {
   session.AutodetectProxyConfiguration //
   session.DirectProxyConfiguration //
   session.ManualProxyConfiguration //
   session.PacProxyConfiguration //
   session.SystemProxyConfiguration
}

session.AutodetectProxyConfiguration = (
   proxyType: "autodetect",
   Extensible
)

session.DirectProxyConfiguration = (
   proxyType: "direct",
   Extensible
)

session.ManualProxyConfiguration = (
   proxyType: "manual",
   ? httpProxy: text,
   ? sslProxy: text,
   ? session.SocksProxyConfiguration,
   ? noProxy: [*text],
   Extensible
)

session.SocksProxyConfiguration = (
   socksProxy: text,
   socksVersion: 0..255,
)

session.PacProxyConfiguration = (
   proxyType: "pac",
   proxyAutoconfigUrl: text,
   Extensible
)

session.SystemProxyConfiguration = (
   proxyType: "system",
   Extensible
)

</pre>

#### The session.UserPromptHandler Type #### {#type-session-UserPromptHandler}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.UserPromptHandler = {
  ? alert: session.UserPromptHandlerType,
  ? beforeUnload: session.UserPromptHandlerType,
  ? confirm: session.UserPromptHandlerType,
  ? default: session.UserPromptHandlerType,
  ? file: session.UserPromptHandlerType,
  ? prompt: session.UserPromptHandlerType,
}
</pre>

The <code>session.UserPromptHandler</code> type represents the configuration of
the user prompt handler.

Note: <code>file</code> handles file picker. "accept" and "dismiss" dismisses
the picker. "ignore" keeps the picker open.

#### The session.UserPromptHandlerType Type #### {#type-session-UserPromptHandlerType}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.UserPromptHandlerType = "accept" / "dismiss" / "ignore";
</pre>

The <code>session.UserPromptHandlerType</code> type represents the behavior
of the user prompt handler.

#### The session.Subscription Type #### {#type-session-Subscription}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
session.Subscription = text
</pre>

The <code>session.Subscription</code> type represents a unique subscription identifier.

#### The session.SubscribeParameters Type #### {#type-session-SubscriptionParameters}

<pre class="cddl" data-cddl-module="remote-cddl">
session.SubscribeParameters = {
  events: [+text],
  ? contexts: [+browsingContext.BrowsingContext],
  ? userContexts: [+browser.UserContext],
}
</pre>

The <code>session.SubscribeParameters</code> type represents a request to
subscribe to a specific set of events.

#### The session.UnsubscribeByIDRequest Type #### {#type-session-UnsubscribeByIDRequest}

<pre class="cddl" data-cddl-module="remote-cddl">
session.UnsubscribeByIDRequest = {
  subscriptions: [+session.Subscription],
}
</pre>

The <code>session.UnsubscribeByIDRequest</code> type represents a request to
remove event subscriptions identified by subscription IDs.

#### The session.UnsubscribeByAttributesRequest Type #### {#type-session-UnsubscribeByAttributesRequest}

<pre class="cddl" data-cddl-module="remote-cddl">
session.UnsubscribeByAttributesRequest = {
  events: [+text],
}
</pre>

The <code>session.UnsubscribeByAttributesRequest</code> type represents a request to
unsubscribe using subscription attributes.

### Commands ### {#module-session-commands}

#### The session.status Command #### {#command-session-status}

The <dfn export for=commands>session.status</dfn> command returns information about
whether a remote end is in a state in which it can create new sessions,
but may additionally include arbitrary meta information that is specific
to the implementation.

This is a [=static command=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      session.Status = (
        method: "session.status",
        params: EmptyParams,
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      session.StatusResult = {
        ready: bool,
        message: text,
      }
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for session.status">

The [=remote end steps=] given <var ignore>session</var>, and <var
ignore>command parameters</var> are:

1. Let |body| be a new [=/map=] with the following properties:

   <dl>
      <dt>"ready"</dt>
      <dd>The [=remote end=]’s [=readiness state=].</dd>

      <dt>"message"</dt>
      <dd>An implementation-defined string explaining the [=remote end=]’s
      [=readiness state=].</dd>
   </dl>

1. Return [=success=] with data |body|

</div>

#### The session.new Command #### {#command-session-new}

The <dfn export for=commands>session.new</dfn> command allows creating a new
[=BiDi session=].

Note: A session created this way will not be accessible via HTTP.

This is a [=static command=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      session.New = (
        method: "session.new",
        params: session.NewParameters
      )

      session.NewParameters = {
        capabilities: session.CapabilitiesRequest
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      session.NewResult = {
        sessionId: text,
        capabilities: {
          acceptInsecureCerts: bool,
          browserName: text,
          browserVersion: text,
          platformName: text,
          setWindowRect: bool,
          userAgent: text,
          ? proxy: session.ProxyConfiguration,
          ? unhandledPromptBehavior: session.UserPromptHandler,
          ? webSocketUrl: text,
          Extensible
        }
      }
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for session.new">

The [=remote end steps=] given |session| and |command parameters| are:

1. If |session| is not null, return an [=error=] with error code [=session not created=].

1. If the implementation is unable to start a new session for any reason, return
   an [=error=] with error code [=session not created=].

1. Let |flags| be a [=/set=] containing "<code>bidi</code>".

1. Let |capabilities json| be the result of [=trying=] to [=process capabilities=]
   with |command parameters| and |flags|.

1. Let |capabilities| be [=convert a JSON-derived JavaScript value to an Infra
   value=] with |capabilities json|.

1. Let |session| be the result of [=trying=] to [=create a session=] with
   |capabilities| and |flags|.

1. Set |session|'s [=BiDi flag=] to true.

   Note: the connection for this session will be set to the current connection
   by the caller.

1. Let |body| be a new [=/map=] matching the <code>session.NewResult</code> production,
   with the <code>sessionId</code> field set to |session|'s [=session ID=], and
   the <code>capabilities</code> field set to |capabilities|.

1. Return [=success=] with data |body|.

</div>

#### The session.end Command #### {#command-session-end}

The <dfn export for=commands>session.end</dfn> command ends the current
[=/session=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      session.End = (
        method: "session.end",
        params: EmptyParams
      )

      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      session.EndResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for session.end">

The [=remote end steps=] given |session| and <var ignore>command parameters</var> are:

1. [=End the session=] with |session|.

1. Return [=success=] with data null, and in parallel run the following steps:

  1. Wait until the [=Send a WebSocket message=] steps have been called with the
     response to this command.

     Issue: this is rather imprecise language, but hopefully it's clear that the
     intent is that we send the response to the command before starting shutdown
     of the connections.

  1. [=Cleanup the session=] with |session|.

</div>


#### The session.subscribe Command #### {#command-session-subscribe}

The <dfn export for=commands>session.subscribe</dfn> command enables certain events
either globally or for a set of navigables.

Issue: This needs to be generalized to work with realms too.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      session.Subscribe = (
        method: "session.subscribe",
        params: session.SubscribeParameters
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        session.SubscribeResult = {
          subscription: session.Subscription,
        }
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for session.subscribe">
The [=remote end steps=] with |session| and |command parameters| are:

1. Let |event names| be an empty [=/set=].

1. For each entry |name| in |command parameters|["<code>events</code>"],
   let |event names| be the [=set/union=] of |event names| and the result of
   [=trying=] to [=obtain a set of event names=] with |name|.

1. Let |input user context ids| be [=set/create|create a set=] with |command parameters|[<code>userContexts</code>].

1. Let |input context ids| be [=set/create|create a set=] with |command parameters|[<code>contexts</code>].

1. If |input user context ids| is not empty and |input context ids| is not empty,
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |subscription navigables| be a [=/set=].

1. Let |top-level traversable context ids| be a [=/set=].

1. If |input context ids| is not empty:

   1. Let |navigables| be the result of [=trying=] to [=get valid navigables by ids=] with |input context ids|.

   1. Set |subscription navigables| be [=get top-level traversables=] with |navigables|.

   1. For each |navigable| in |subscription navigables|:

      1. [=set/Append=] |navigable|'s [=navigable id=] to |top-level traversable context ids|.

1. Otherwise, if |input user context ids| is not empty:

    1. [=list/For each=] |user context id| of |input user context ids|:

       1. Let |user context| be [=get user context=] with |user context id|.

       1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

       1. [=list/For each=] |top-level traversable| in the list of all [=/top-level traversables=]
          whose [=associated user context=] is |user context|:

          1. [=list/Append=] |top-level traversable| to |subscription navigables|.

1. Otherwise, set |subscription navigables| to a [=/set=] of all [=navigable/top-level traversables=] in the [=remote end=].

1. Let |subscription| be a [=subscription=] with
   [=subscription/subscription id=] set to the string representation of a [[!RFC9562|UUID]],
   [=subscription/event names=] set to |event names|,
   [=subscription/top-level traversable ids=] set to |top-level traversable context ids| and
   [=subscription/user context ids=] set to |input user context ids|.

1. Let |subscribe step events| be a new [=/map=].

1. For each |event name| in the |event names|:

   1. If the [=event=] with [=event name=] |event name| does not define [=remote end
     subscribe steps=], continue;

   1. Let |existing navigables| be a [=set of top-level traversables for which an event is enabled=] with |session| and |event name|.

   1. Set |subscribe step events|[|event name|] to [=set/difference=] of
      |subscription navigables| and |existing navigables|.

1. Append |subscription| to |session|'s [=subscriptions=].

1. Append |subscription|'s [=subscription/subscription id=] to |session|'s [=known subscription ids=].

1. [=map/Sort in ascending order=] |subscribe step events| using the following less
   than algorithm given two entries with keys |event name one| and |event
   name two|:

    1. Let |event one| be the [=event=] with name |event name one|

    1. Let |event two| be the [=event=] with name |event name two|

    1. Return true if |event one|'s [=subscribe priority=] is less than |event
       two|'s subscribe priority, or false otherwise.

1. If |subscription| is [=subscription/global=], let |include global| be true, otherwise let
   |include global| be false.

1. For each |event name| → |navigables| in |subscribe step events|:

  1. Run the [=remote end subscribe steps=] for the [=event=] with [=event name=]
     |event name| given |session|, |navigables| and |include global|.

1. Let |body| be a new [=/map=] matching the <code>session.SubscribeResult</code> production,
   with the <code>subscription</code> field set to |subscription|'s [=subscription/subscription id=].

1. Return [=success=] with data |body|.

</div>

#### The session.unsubscribe Command #### {#command-session-unsubscribe}

The <dfn export for=commands>session.unsubscribe</dfn> command disables events
either globally or for a set of navigables.

Issue: This needs to be generalised to work with realms too.

<dl>
   <dt>Command Type</dt>
   <dd>
     <pre class="cddl" data-cddl-module="remote-cddl">
     session.Unsubscribe = (
       method: "session.unsubscribe",
       params: session.UnsubscribeParameters,
     )

     session.UnsubscribeParameters = session.UnsubscribeByAttributesRequest / session.UnsubscribeByIDRequest
     </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      session.UnsubscribeResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for session.unsubscribe">
The [=remote end steps=] with |session| and |command parameters| are:

1. If |command parameters| does not [=map/contain=] "<code>subscriptions</code>":

   Note: The condition implies that |command parameters| is matching the session.UnsubscribeByAttributesRequest production.

   1. Let |event names| be an empty [=/set=].

   1. For each entry |name| in |command parameters|["<code>events</code>"],
      let |event names| be the [=set/union=] of |event names| and the result of
      [=trying=] to [=obtain a set of event names=] with |name|.

   1. Let |new subscriptions| to be a [=/list=].

   1. Let |matched events| to be a [=/set=].

   1. For each |subscription| of |session|'s [=subscriptions=]:

      1. If [=set/intersection=] of |subscription|'s [=subscription/event names=] and |event names| is an empty [=/set=]:

         1. [=list/append=] |subscription| to |new subscriptions|.

         1. [=Continue=].

      1. If |subscription| is not [=subscription/global=]:

         1. [=list/append=] |subscription| to |new subscriptions|.

         1. [=Continue=].

      1. Let |subscription event names| be [=set/clone=] of |subscription|'s [=subscription/event names=].

      1. For each |event name| of |event names|:

         1. If |subscription event names| [=set/contains=] |event name|:

            1. [=list/Append=] |event name| to |matched events|.

            1. [=list/Remove=] |event name| from |subscription event names|.

      1. If |subscription event names| is not empty:

         1. Let |cloned subscription| be a [=subscription=] with
            [=subscription/subscription id=] set to |subscription|'s [=subscription/subscription id=],
            [=subscription/event names=] set to a new [=/set=] containing |subscription event names|.

         1. [=list/append=] |cloned subscription| to |new subscriptions|.

   1. If |matched events| is not [=set/equal=] to |event names|, return [=error=] with [=error code=] [=invalid argument=].

   1. Set |session|'s [=subscriptions=] to |new subscriptions|.

1. Otherwise:

   1. Let |subscriptions| be [=set/create|create a set=] with |command parameters|[<code>subscriptions</code>].

   1. Let |unknown subscription ids| to [=set/difference=] between |subscriptions| and |session|'s [=known subscription ids=].

   1. If |unknown subscription ids| is not empty:

      1. Return [=error=] with [=error code=] [=invalid argument=].

   1. Let |subscriptions to remove| be an empty [=/set=].

   1. For each |subscription| in |session|'s [=subscriptions=]:

      1. If |subscriptions| [=set/contains=] |subscription|'s [=subscription/subscription id=]:

         1. [=set/Append=] |subscription| to |subscriptions to remove|.

   1. Set |session|'s [=known subscription ids=] to
      [=set/difference=] between |session|'s [=known subscription ids=] and |subscriptions|.

   1. [=list/Remove=] each item in |subscriptions to remove| from |session|'s [=subscriptions=].

1. Return [=success=] with data null.

</div>

## The browser Module ## {#module-browser}

The <dfn export for=modules>browser</dfn> module contains commands for
managing the remote end browser process.

### Definition ### {#module-browser-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
BrowserCommand = (
  browser.Close //
  browser.CreateUserContext //
  browser.GetClientWindows //
  browser.GetUserContexts //
  browser.RemoveUserContext //
  browser.SetClientWindowState //
  browser.SetDownloadBehavior
)
</pre>

{^local end definition^}

<!-- Nothing yet -->
<pre class="cddl" data-cddl-module="local-cddl">
BrowserResult = (
  browser.CloseResult /
  browser.CreateUserContextResult /
  browser.GetClientWindowsResult /
  browser.GetUserContextsResult /
  browser.RemoveUserContextResult /
  browser.SetClientWindowStateResult /
  browser.SetDownloadBehaviorResult
)
</pre>

### Windows ### {#module-browser-window}

<!-- This should probably be defined in CSSOM-View -->

Each [=/top-level traversable=] is associated with a single <dfn>client
window</dfn> which represents a rectangular area containing the
<a spec=css2>viewport</a> that will be used to render that [=/top-level
traversable=]'s [=active document=] when its [=visibility state=] is
"<code>visible</code>", as well as any browser-specific user interface elements
associated with displaying the traversable (e.g. any URL bar, toolbars, or OS
window decorations).

A [=client window=] has a <dfn>client window id</dfn> which is a string uniquely
identifying that window.

A [=client window=] has an <dfn for="client window">x-coordinate</dfn>, which is
the number of CSS pixels between the left edge of the [=web-exposed screen
area=] and the left edge of the window, or zero if that doesn't make sense for a
particular window.

A [=client window=] has a <dfn for="client window">y-coordinate</dfn>, which is
the number of CSS pixels between the top edge of the [=web-exposed screen
area=] and the top edge of the window, or zero if that doesn't make sense for a
particular window.

A [=client window=] has a <dfn for="client window">width</dfn>, which is the
width of the window's rectangle in CSS pixels.

A [=client window=] has a <dfn for="client window">height</dfn>, which is the
height of the window's rectangle in CSS pixels.

To <dfn>maximize the client window</dfn> |window| an implementation should either
perform steps corresponding to the platform notion of maximizing |window|, or
position |window| such that its [=client window/x-coordinate=] is as close as
possible to 0, its [=client window/y-coordinate=] is as close as possible to 0,
its [=client window/width=] is as close as possible to the width of the
[=web-exposed screen area=] and its [=client window/height=] is as close as possible
to the height of the [=web-exposed screen area=]. If either of these options are
supported then <dfn>maximize client window is supported</dfn>.

To <dfn>minimize the client window</dfn> |window| an implementation should either
perform steps corresponding to the platform notion of minimizing |window|, or
otherwise hide |window| such that all the [=active documents=] in
[=/top-level traversables=] associated with |window| have [=visibility state=]
"<code>hidden</code>" and |window|'s [=client window/width=] and [=client
window/height=] are both as close as possible to 0. If either of these options
are supported then <dfn>minimize client window is supported</dfn>.

To <dfn>restore the client window</dfn> |window| an implementation should ensure that it's
neither in a platform-defined maximized state, nor in a platform-defined
minimized state, and that if there is one or more [=/top-level traversable=]
associated with |window|, at least one of those has an [=active document=] in
the "<code>visible</code>" state. If this is supported then <dfn>restore client
window is supported</dfn>.

<div algorithm>

To <dfn>get the client window state</dfn> given |window|:

1. Let |documents| be an empty [=/list=].

1. Let |visible documents| be an empty [=/list=].

1. For each [=/top-level traversable=] |traversable|:

  1. If |traversable|'s [=client window=] is not |window| then continue.

  1. Let |document| be |traversable|'s [=active document=].

  1. [=list/Append=] |document| to |documents|.

  1. If |document|'s [=visibility state=] is "<code>visible</code>",
     [=list/Append=] |document| to |visible documents|.

1. For each |document| in |visible documents|:

  1. If |document|'s [=fullscreen element=] is not null, return
     "<code>fullscreen</code>".

1. If |visible documents| is [=list/empty=] but |documents| is not
   [=list/empty=], or if |window| is otherwise in an OS-specific minimized
   state, return "<code>minimized</code>".

   Note: This will usually, but not necessarily, mean that |window|'s [=client
   window/width=] and [=client window/height=] are equal to 0.

1. If |window| is in an OS-specific maximized state return
   "<code>maximized</code>".

   Note: This will usually, but not necessarily, mean that |window|'s [=client
   window/width=] is equal to the width of the [=web-exposed screen area=] and
   |window|'s [=client window/height=] is equal to the height of the
   [=web-exposed screen area=].

1. Return "<code>normal</code>".

</div>

<div algorithm>

To <dfn>set the client window state</dfn> given |window| and |state|:

1. Let |current state| be [=get the client window state=] with |window|.

1. If |current state| is "<code>fullscreen</code>", "<code>maximized</code>", or "<code>minimized</code>" and is equal to |state|, return [=success=] with data null.

1. In the following list of conditions and associated steps, run the first set of steps for which the associated condition is true:

   <dl>
    <dt>"<code>fullscreen</code>"
    <dd>If not [=fullscreen is supported=] return [=error=] with [=error code=]
    [=unsupported operation=].

    <dt>"<code>normal</code>"
    <dd>If not [=restore client window is supported=] for |window| return [=error=] with
    [=error code=] [=unsupported operation=].

    <dt>"<code>maximize</code>"
    <dd>If not [=maximize client window is supported=] for |window| return [=error=] with
    [=error code=] [=unsupported operation=].

    <dt>"<code>minimize</code>"
    <dd>If not [=minimize client window is supported=] for |window| return [=error=] with
    [=error code=] [=unsupported operation=].
   </dl>

1. Let |documents| be an empty [=/list=].

1. For each [=/top-level traversable=] |traversable|:

  1. If |traversable|'s associated [=client window=] is not |window| then
     continue.

  1. Let |document| be |traversable|'s [=active document=].

  1. Append |document| to |documents|.

1. If |documents| is [=list/empty=] return [=error=] with [=error code=]
   [=no such client window=].

1. If |current state| is "<code>fullscreen</code>":

  1. For each |document| in |documents|:

    1. [=Fully exit fullscreen=] with |document|.

       Note: This is a no-op for documents in window that are not fullscreen.

1. If |current state| is "<code>maximized</code>" or "<code>minimized</code>":

  1. [=Restore the client window=] |window|.

1. Switch on the value of |state|:

   <dl>
    <dt>"<code>fullscreen</code>"
    <dd>
      1. For each |document| in |documents|:

        1. If |document|'s [=visibility state=] is "<code>visible</code>",
           [=fullscreen an element=] with |document|'s [=document element=].

        1. [=Break=].

    <dt>"<code>maximize</code>"
    <dd>1. [=Maximize the client window=] |window|.

    <dt>"<code>minimize</code>"
    <dd>1. [=Minimize the client window=] |window|.

  </dl>

1. Return [=success=] with data null.

</div>

### Types ### {#module-browser-types}

#### The browser.ClientWindow Type #### {#type-browser-ClientWindow}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browser.ClientWindow = text;
</pre>

The <code>browser.ClientWindow</code> uniquely identifies a [=client window=].

#### The browser.ClientWindowInfo Type #### {#type-browser-ClientWindowInfo}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browser.ClientWindowInfo = {
  active: bool,
  clientWindow: browser.ClientWindow,
  height: js-uint,
  state: "fullscreen" / "maximized" / "minimized" / "normal",
  width: js-uint,
  x: js-int,
  y: js-int,
}
</pre>

The <code>browser.ClientWindowInfo</code> type represents properties of a
[=client window=].

<div algorithm>
To <dfn>get the client window info</dfn> given |client window|:

1. Let |client window id| be the [=client window id=] for |client window|.

1. Let |state| be [=get the client window state=] with |client window|.

1. If |client window| can receive keyboard input channeled from the operating
   system, let |active| be true, otherwise let |active| be false.

   Note: This could mean that a [=/top-level traversable=] whose [=client
   window=] is |client window| has [=system focus=], or it could mean that the
   user interface of the browser itself currently has focus.

1. Let |client window info| be a [=/map=] matching the
   <code>browser.ClientWindowsInfo</code> production with the
   <code>clientWindow</code> field set to |client window id|, <code>state</code>
   field set to |state|, the <code>x</code> field set to |client window|'s
   [=client window/x-coordinate=], the <code>y</code> field set to |client
   window|'s [=client window/y-coordinate=], the <code>width</code> field set to
   |client window|'s [=client window/width=], the <code>height</code> field
   set to |client window|'s [=client window/height=], and the <code>active</code>
   field set to |active|.

1. Return |client window info|

</div>

#### The browser.UserContext Type #### {#type-browser-UserContext}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browser.UserContext = text;
</pre>

The <code><dfn>browser.UserContext</dfn></code> unique identifies a [=user context=].

#### The browser.UserContextInfo Type #### {#type-browser-UserContextInfo}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browser.UserContextInfo = {
  userContext: browser.UserContext
}
</pre>

The <code>browser.UserContextInfo</code> type represents properties of a [=user
context=].

### Commands ### {#module-browser-commands}

#### The browser.close Command #### {#command-browser-close}

The <dfn export for=commands>browser.close</dfn> command terminates all
WebDriver sessions and cleans up automation state in the remote browser instance.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.Close = (
        method: "browser.close",
        params: EmptyParams,
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browser.CloseResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.close">
The [=remote end steps=] with |session| and <var ignore>command parameters</var> are:

1. [=End the session=] with |session|.

1. If [=active sessions=] is not [=list/empty=] an implementation may
   return [=error=] with [=error code=] [=unable to close browser=], and then
   run the following steps [=in parallel=]:

    1. Wait until the [=Send a WebSocket message=] steps have been called with the
       response to this command.

    1. [=Cleanup the session=] with |session|.

   Note: The behaviour in cases where the browser has multiple automation
   sessions is currently unspecified. It might be that any session can close the
   browser, or that only the final open session can actually close the browser,
   or only the first session started can. This behaviour might be fully
   specified in a future version of this specification.

1. For each |active session| in [=active sessions=]:

  1. [=End the session=] |active session|.

  1. [=Cleanup the session=] with |active session|

1. Return [=success=] with data null, and run the following steps [=in parallel=].

  1. Wait until the [=Send a WebSocket message=] steps have been called with the
     response to this command.

  1. [=Cleanup the session=] with |session|.

  1. [=Close=] any [=navigable/top-level traversables=] without [=prompting to unload=].

  1. Perform implementation defined steps to clean up resources associated with
     the [=remote end=] under automation.

     Note: For example this might include cleanly shutting down any OS-level
     processes associated with the browser under automation, removing temporary
     state, such as user profile data, created by the [=remote end=] while under
     automation, or shutting down the [=WebSocket Listener=]. Because of
     differences between browsers and operating systems it is not possible to
     specify in detail precise invariants [=local ends=] can depend on here.

</div>

#### The browser.createUserContext Command #### {#command-browser-createUserContext}

The <dfn export for=commands>browser.createUserContext</dfn> command creates a
[=user context=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.CreateUserContext = (
        method: "browser.createUserContext",
        params: browser.CreateUserContextParameters,
      )

      browser.CreateUserContextParameters = {
        ? acceptInsecureCerts: bool,
        ? proxy: session.ProxyConfiguration,
        ? unhandledPromptBehavior: session.UserPromptHandler
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browser.CreateUserContextResult = browser.UserContextInfo
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.createUserContext">

The [=remote end steps=] with |session| and |command parameters| are:

1. Let |user context| be a new [=user context=].

1. If |command parameters| [=map/contain=] "<code>acceptInsecureCerts</code>":

  Note: If "<code>acceptInsecureCerts</code>" is set, it overrides the
  [=accept insecure TLS=] flag's behavior.

   1. Let |acceptInsecureCerts| be
      |command parameters|["<code>acceptInsecureCerts</code>"]:

   1. If |acceptInsecureCerts| is true and [=endpoint node=] doesn't support
      accepting insecure TLS connections, return [=error=] with [=error code=]
      [=unsupported operation=].

   1. [=map/Set=] |session|'s
      [=user context to accept insecure certificates override map=][|user context|]
      to |acceptInsecureCerts|.

1. If |command parameters| [=map/contains=] "<code>unhandledPromptBehavior</code>",
   [=map/set=] [=unhandled prompt behavior overrides map=][|user context|] to
   |command parameters|["<code>unhandledPromptBehavior</code>"].

1. If |command parameters| [=map/contains=] "<code>proxy</code>":

   1. Let |proxy configuration| be |command parameters|["<code>proxy</code>"].

   1. If the [=remote end=] is unable to configure proxy settings per [=user context=],
       or is unable to configure the proxy with |proxy configuration|, return  [=error=] with
       [=error code=] [=unsupported operation=].

   1. [=map/Set=] |session|'s
      [=user context to proxy configuration map=][|user context|]
      to |proxy configuration|.

1. [=set/Append=] |user context| to the [=set of user contexts=].

1. Let |user context info| be a [=/map=] matching the
    <code>browser.UserContextInfo</code> production with the
    <code>userContext</code> field set to |user context|'s [=user context id=].

1. Return [=success=] with data |user context info|.

</div>

#### The browser.getClientWindows Command #### {#command-browser-getClientWindows}

The <dfn export for=commands>browser.getClientWindows</dfn> command returns a
list of [=client window=]s.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.GetClientWindows = (
        method: "browser.getClientWindows",
        params: EmptyParams,
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browser.GetClientWindowsResult = {
        clientWindows: [ * browser.ClientWindowInfo]
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.getClientWindows">

The [=remote end steps=] are:

1. Let |client window ids| be an empty [=/set=].

1. Let |client windows| be an empty [=/list=].

1. For each [=/top-level traversable=] |traversable|:

  1. Let |client window| be |traversable|'s associated [=client window=]

  1. Let |client window id| be the [=client window id=] for |client window|.

  1. If |client window ids| [=set/contains=] |client window id|, continue.

  1. [=set/Append=] |client window id| to |client window ids|.

  1. Let |client window info| be [=get the client window info=] with |client
     window|.

  1. [=list/Append=] |client window info| to |client windows|.

1. Let |result| be a [=/map=] matching the
   <code>browser.GetClientWindowsResult</code> production with the
   <code>clientWindows</code> field set to |client windows|.

1. Return [=success=] with data |result|.

</div>


#### The browser.getUserContexts Command #### {#command-browser-getUserContexts}

The <dfn export for=commands>browser.getUserContexts</dfn> command returns a
list of [=user context=]s.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.GetUserContexts = (
        method: "browser.getUserContexts",
        params: EmptyParams,
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browser.GetUserContextsResult = {
        userContexts: [ + browser.UserContextInfo]
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.getUserContexts">

The [=remote end steps=] are:

1. Let |user contexts| be an empty [=/list=].

1. For each |user context| in the [=set of user contexts=]:

  1. Let |user context info| be a [=/map=] matching the
     <code>browser.UserContextInfo</code> production with the
     <code>userContext</code> field set to |user context|'s [=user context id=].

  1. [=list/Append=] |user context info| to |user contexts|.

1. Let |result| be a [=/map=] matching the
   <code>browser.GetUserContextsResult</code> production with the
   <code>userContexts</code> field set to |user contexts|.

1. Return [=success=] with data |result|.

</div>


#### The browser.removeUserContext Command #### {#command-browser-removeUserContext}

The <dfn export for=commands>browser.removeUserContext</dfn> command closes a
user context and all navigables in it without running
<code>beforeunload</code> handlers.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.RemoveUserContext = (
        method: "browser.removeUserContext",
        params: browser.RemoveUserContextParameters
      )

      browser.RemoveUserContextParameters = {
        userContext: browser.UserContext
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browser.RemoveUserContextResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.removeUserContext">

The [=remote end steps=] with |command parameters| are:

1. Let |user context id| be |command  parameters|["<code>userContext</code>"].

1. If |user context id| is <code>"default"</code>, return [=error=] with [=error
   code=] [=invalid argument=].

1. Set |user context| to [=get user context=] with |user context id|.

1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

1. For each [=navigable/top-level traversable=] |navigable|:

  1. If |navigable|'s [=associated user context=] is |user context|:

    1. [=Close=] |navigable| without [=prompting to unload=].

1. [=set/Remove=] |user context| for the [=set of user contexts=].

1. Return [=success=] with data null.

</div>

#### The browser.setClientWindowState Command #### {#command-browser-setClientWindowState}

The <dfn export for=commands>browser.setClientWindowState</dfn> command sets the
dimensions of a [=client window=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.SetClientWindowState = (
        method: "browser.setClientWindowState",
        params: browser.SetClientWindowStateParameters
      )

      browser.SetClientWindowStateParameters = {
        clientWindow: browser.ClientWindow,
        (browser.ClientWindowNamedState // browser.ClientWindowRectState)
      }

      browser.ClientWindowNamedState = (
        state: "fullscreen" / "maximized" / "minimized"
      )

      browser.ClientWindowRectState = (
        state: "normal",
        ? width: js-uint,
        ? height: js-uint,
        ? x: js-int,
        ? y: js-int,
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browser.SetClientWindowStateResult = browser.ClientWindowInfo
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browser.setClientWindowState">

The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. If the implementation does not support setting the client window state at
   all, then return [=error=] with [=error code=] [=unsupported operation=].

1. If there is a [=client window=] with [=client window id=] |command
   parameters|["<code>clientWindow</code>"], let |client window| be that [=client
   window=]. Otherwise return [=error=] with [=error code=] [=no such client window=].

1. [=Try=] to [=set the client window state=] with |client window| and |command
   parameters|["<code>state</code>"].

1. If |command parameters|["<code>state</code>"] is "<code>normal</code>":

  1. If |command parameters| [=map/contains=] "<code>x</code>" and the
     implementation supports positioning [=client windows=], set the [=client
     window/x-coordinate=] of |client window| to a value that is as close as
     possible |command parameters|["<code>x</code>"].

   1. If |command parameters| [=map/contains=] "<code>y</code>" and the
     implementation supports positioning [=client windows=], set the [=client
     window/y-coordinate=] of |client window| to a value that is as close as
     possible |command parameters|["<code>y</code>"].

   1. If |command parameters| [=map/contains=] "<code>width</code>" and the
     implementation supports resizing [=client windows=], set the [=client
     window/width=] of |client window| to a value that is as close as possible
     |command parameters|["<code>width</code>"].

   1. If |command parameters| [=map/contains=] "<code>width</code>" and the
     implementation supports resizing [=client windows=], set the [=client
     window/width=] of |client window| to a value that is as close as possible
     |command parameters|["<code>width</code>"].

1. Let |client window info| be [=get the client window info=] with |client window|.

1. Return [=success=] with data |client window info|.

Note: For simplicity this models all client window operations as
synchronous. Therefore the returned client window dimensions are expected to be
those after the window has reached its new state.

</div>

#### The browser.setDownloadBehavior Command #### {#command-browser-setDownloadBehavior}

A <dfn>download behavior struct</dfn> is a [=struct=] with:
* [=struct/item=] named <dfn id="download-behavior-struct-allowed" for="download-behavior-struct">allowed</dfn> which is a boolean;
* [=struct/item=] named <dfn id="download-behavior-struct-destination-folder" for="download-behavior-struct">destinationFolder</dfn> which is a string or null.

A [=remote end=] has a <dfn>download behavior</dfn> which is a [=struct=] with an [=struct/item=]
named <dfn for="download behavior">default download behavior</dfn>, which is a
[=download behavior struct=] or null, and an [=struct/item=] named <dfn
for="download behavior">user context download behavior</dfn>, which is a weak map between
[=user context|user contexts=] and [=download behavior struct=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browser.SetDownloadBehavior = (
        method: "browser.setDownloadBehavior",
        params: browser.SetDownloadBehaviorParameters
      )

      browser.SetDownloadBehaviorParameters = {
        downloadBehavior: browser.DownloadBehavior / null,
        ? userContexts: [+browser.UserContext]
      }

      browser.DownloadBehavior = {
        (
          browser.DownloadBehaviorAllowed //
          browser.DownloadBehaviorDenied
        )
      }

      browser.DownloadBehaviorAllowed = (
        type: "allowed",
        destinationFolder: text
      )

      browser.DownloadBehaviorDenied = (
        type: "denied"
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browser.SetDownloadBehaviorResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
To <dfn export>get download behavior</dfn> given |navigable|:

1. Let |user context| be |navigable|'s [=associated user context=].

1. If [=download behavior=]'s [=download behavior/user context download behavior=] [=map/contains=]
   |user context|, return [=download behavior=]'s
   [=download behavior/user context download behavior=][|user context|].

1. Return [=download behavior=]'s [=download behavior/default download behavior=].

</div>

<div algorithm="remote end steps for browser.setDownloadBehavior">

The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. If |command parameters|["<code>downloadBehavior</code>"] is null, let
   |download behavior| be null.

1. Otherwise:

   1. If |command parameters|["<code>downloadBehavior</code>"]["<code>type</code>"]
      is "<code>allowed</code>", let |allowed| be true, otherwise let |allowed| be
      false.

   1. If |command parameters|["<code>downloadBehavior</code>"] [=map/contains=]
      "<code>destinationFolder</code>", let |destinationFolder| be
      |command parameters|["<code>downloadBehavior</code>"]["<code>destinationFolder</code>"],
      otherwise let |destinationFolder| be null.

   1. Let |download behavior| be a [=download behavior struct=] with
      [=download-behavior-struct/allowed=] set to |allowed| and
      [=download-behavior-struct/destinationFolder=] set to |destinationFolder|.

1. If the implementation does not support required |download behavior|, then return
   [=error=] with [=error code=] [=unsupported operation=].

1. If the <code>userContexts</code> field of |command parameters| is present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |download behavior| is null, [=map/remove=] |user context| from [=download behavior=]'s
         [=download behavior/user context download behavior=].

      1. Otherwise, [=map/set=] [=download behavior=]'s
         [=download behavior/user context download behavior=][|user context|] to
         |download behavior|.

1. Otherwise, set [=download behavior=]'s [=download behavior/default download behavior=] to
   |download behavior|.

1. Return [=success=] with data null.

</div>

## The browsingContext Module ## {#module-browsingContext}

The <dfn export for=modules>browsingContext</dfn> module contains commands and
events relating to [=/navigables=].

Note: For historic reasons this module is called <code>browsingContext</code>
rather than <code>navigable</code>, and the protocol uses the term
<code>context</code> to refer to navigables, particularly as a field in command
and response parameters.

The progress of navigation is communicated using an immutable [=struct=]
<dfn export>WebDriver BiDi navigation status</dfn>, which has the following [=struct/item|items=]:

<dl>
  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-id">id</dfn></dt>
  <dd>The [=navigation id=] for the navigation, or null when the navigation is
  canceled before making progress.</dd>

  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-status">status</dfn></dt>
    <dd>A status code that is either
      "<dfn export id="navigation-status-canceled"><code>canceled</code></dfn>",
      "<dfn export id="navigation-status-pending"><code>pending</code></dfn>", or
      "<dfn export id="navigation-status-complete"><code>complete</code></dfn>".
    </dd>

  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-url">url</dfn></dt>
  <dd>The URL which is being loaded in the navigation</dd>

  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-suggested-filename">suggestedFilename</dfn></dt>
  <dd>If the navigation is a download, suggested filename, otherwise null.</dd>

  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-downloaded-filepath">downloadedFilepath</dfn></dt>
  <dd>If the navigation is a download which is finished and the downloaded file is
  available, absolute filepath of the downloaded file, otherwise null.</dd>

  <dt><dfn export for="WebDriver BiDi navigation status" id="navigation-status-download-response">downloadResponse</dfn></dt>
  <dd>If the navigation is a download, [=response=], otherwise null.</dd>
</dl>

### Definition ### {#module-browsingContext-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
BrowsingContextCommand = (
  browsingContext.Activate //
  browsingContext.CaptureScreenshot //
  browsingContext.Close //
  browsingContext.Create //
  browsingContext.GetTree //
  browsingContext.HandleUserPrompt //
  browsingContext.LocateNodes //
  browsingContext.Navigate //
  browsingContext.Print //
  browsingContext.Reload //
  browsingContext.SetBypassCSP //
  browsingContext.SetViewport //
  browsingContext.StartScreencast //
  browsingContext.StopScreencast //
  browsingContext.TraverseHistory
)
</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
BrowsingContextResult = (
  browsingContext.ActivateResult /
  browsingContext.CaptureScreenshotResult /
  browsingContext.CloseResult /
  browsingContext.CreateResult /
  browsingContext.GetTreeResult /
  browsingContext.HandleUserPromptResult /
  browsingContext.LocateNodesResult /
  browsingContext.NavigateResult /
  browsingContext.PrintResult /
  browsingContext.ReloadResult /
  browsingContext.SetBypassCSPResult /
  browsingContext.SetViewportResult /
  browsingContext.StartScreencastResult /
  browsingContext.StopScreencastResult /
  browsingContext.TraverseHistoryResult
)

BrowsingContextEvent = (
  browsingContext.ContextCreated //
  browsingContext.ContextDestroyed //
  browsingContext.DomContentLoaded //
  browsingContext.DownloadEnd //
  browsingContext.DownloadWillBegin //
  browsingContext.FragmentNavigated //
  browsingContext.HistoryUpdated //
  browsingContext.Load //
  browsingContext.NavigationAborted //
  browsingContext.NavigationCommitted //
  browsingContext.NavigationFailed //
  browsingContext.NavigationStarted //
  browsingContext.UserPromptClosed //
  browsingContext.UserPromptOpened
)
</pre>

A [=remote end=] has a <dfn>device pixel ratio overrides</dfn> which is a weak map
between [=/navigables=] and device pixel ratio overrides. It is initially empty.

Note: this map is not cleared when the final session ends i.e. device pixel
ratio overrides outlive any WebDriver session.

A <dfn>viewport dimensions</dfn> is a [=struct=] with:
* [=struct/Item=] named <dfn for="viewport-dimensions">height</dfn> which is an integer;
* [=struct/Item=] named <dfn for="viewport-dimensions">width</dfn> which is an integer.

A <dfn>viewport configuration</dfn> is a [=struct=] with:
* [=struct/Item=] named <dfn for="viewport-configuration">viewport</dfn> which is a [=viewport dimensions=] or null;
* [=struct/Item=] named <dfn for="viewport-configuration">devicePixelRatio</dfn> which is a float or null.

An <dfn>unhandled prompt behavior struct</dfn> is a [=struct=] with:
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-alert">alert</dfn> which is a string or null;
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-beforeUnload">beforeUnload</dfn> which is a string or null;
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-confirm">confirm</dfn> which is a string or null;
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-default">default</dfn> which is a string or null;
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-file">file</dfn> which is a string or null;
* [=struct/Item=] named <dfn attribute for="unhandled-prompt-behavior-prompt">prompt</dfn> which is a string or null.

A [=remote end=] has a <dfn>viewport overrides map</dfn> which is a weak map between [=user context|user contexts=] and [=viewport configuration=].

A [=remote end=] has a <dfn>locale overrides map</dfn> which is a weak map between
[=/navigables=] or [=user context|user contexts=] and string.

A <dfn>screen settings</dfn> is a [=struct=] with an [=struct/item=] named
<dfn attribute for="screen settings">height</dfn> which is an integer,
an [=struct/item=] named <dfn attribute for="screen settings">width</dfn> which is an integer,
an [=struct/item=] named <dfn attribute for="screen settings">x</dfn> which is an integer,
an [=struct/item=] named <dfn attribute for="screen settings">y</dfn> which is an integer.

A [=remote end=] has a <dfn>screen settings overrides</dfn> which is a [=struct=] with
an [=struct/item=] named <dfn for="screen settings overrides">user context screen settings</dfn>,
which is a weak map between [=user context|user contexts=] and [=screen settings=],
and an [=struct/item=] named <dfn for="screen settings overrides">navigable screen settings</dfn>,
which is a weak map between [=/navigables=] and [=screen settings=].

A [=remote end=] has a <dfn>timezone overrides map</dfn> which is a weak map between
[=/navigables=] or [=user context|user contexts=] and string.

A [=remote end=] has an <dfn>unhandled prompt behavior overrides map</dfn> which is a
weak map between [=user context|user contexts=] and [=unhandled prompt behavior struct=].

A [=remote end=] has a <dfn>scripting enabled overrides map</dfn> which is a weak
map between [=/navigables=] or [=user context|user contexts=] and boolean.

A [=remote end=] has a <dfn>download id map</dfn> which is is a weak map between
[=response=] and download ids. It is initially empty.

A <dfn>screencast stream</dfn> is an abstract stream of the viewport of a
[=/top-level traversable=], consisting of a <dfn for="screencast stream">video track</dfn>
containing the rendered visual output of the [=/top-level traversable=]'s document's viewport,
and optionally an <dfn for="screencast stream">audio track</dfn> containing the audio output
of the [=/top-level traversable=]'s document.

A [=BiDi session=] has a <dfn>screencast recordings map</dfn> which is a [=/map=] in
which the keys are [[!RFC9562|UUID]]s, and the values are <dfn>screencast recording</dfn>,
which is a [=struct=] with
an [=struct/item=] named <dfn for="screencast recording">stream</dfn>, which is a [=screencast stream=],
an [=struct/item=] named <dfn for="screencast recording">path</dfn>, which is a string,
an [=struct/item=] named <dfn for="screencast recording">state</dfn>, which is one of
"<code>recording</code>", "<code>stopping</code>", "<code>stopped</code>",
an [=struct/item=] named <dfn for="screencast recording">writeError</dfn>, which is a string or null.

<div algorithm>
To <dfn>start a screencast recording</dfn> given a [=screencast recording=] |recording| and |mime type|:

1. Run the following steps [=in parallel=]:

   1. Begin encoding |recording|'s [=screencast recording/stream=] using |mime type|,
      producing successive chunks of encoded data as [=byte sequences=].
      Produce a new chunk at the implementation defined interaval while
      |recording|'s [=screencast recording/state=] is "<code>recording</code>".

   1. For each chunk |bytes| produced for |recording|, run the following steps:

      1. Append |bytes| to the file at |recording|'s [=screencast recording/path=].
         If this fails:

         1. Set |recording|'s [=screencast recording/writeError=] to an
            implementation-defined string describing the write failure.

         1. [=Stop a screencast recording=] given |recording|.

</div>

<div algorithm>
To <dfn>stop a screencast recording</dfn> given a [=screencast recording=] |recording|:

1. If |recording|'s [=screencast recording/state=] is not "<code>recording</code>" then return.

1. Set |recording|'s [=screencast recording/state=] to "<code>stopping</code>".

1. Stop producing new chunks for |recording|, flush any remaining encoded
   data as a final chunk (processed as in [=start a screencast recording=]),
   stop capturing from |recording|'s [=screencast recording/stream=] and release its
   [=screencast stream/video track=] and, if present, its
   [=screencast stream/audio track=], and then set |recording|'s
   [=screencast recording/state=] to "<code>stopped</code>".

1. Wait until |recording|'s [=screencast recording/state=] is "<code>stopped</code>".

</div>

### Types ### {#module-browsingcontext-types}

#### The browsingContext.BrowsingContext Type #### {#type-browsingContext-Browsingcontext}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browsingContext.BrowsingContext = text;
</pre>

Each [=/navigable=] has an associated <dfn export>navigable id</dfn>,
which is a string uniquely identifying that navigable. This is
implicitly set when the navigable is created. For navigables with an
associated WebDriver [=window handle=] the [=/navigable id=] must be the
same as the [=window handle=].

Each [=/navigable=] also has an <dfn>associated storage partition</dfn>,
which is the [=storage partition=] it uses to persist data.

Each [=/navigable=] also has an associated <dfn>original opener</dfn>,
which is a [=/navigable=] that caused the navigable to open
or null, initially set to null.

<div algorithm>
To <dfn export>get a navigable</dfn> given |navigable id|:

1. If |navigable id| is null, return [=success=] with data null.

1. If there is no navigable with [=navigable id=] |navigable id| return
   [=error=] with [=error code=] [=no such frame=]

1. Let |navigable| be the [=/navigable=] with id |navigable id|.

1. Return [=success=] with data |navigable|.

</div>

#### The browsingContext.Info Type #### {#type-browsingContext-Info}

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
browsingContext.InfoList = [*browsingContext.Info]

browsingContext.Info = {
  children: browsingContext.InfoList / null,
  clientWindow: browser.ClientWindow,
  context: browsingContext.BrowsingContext,
  originalOpener: browsingContext.BrowsingContext / null,
  url: text,
  userContext: browser.UserContext,
  ? parent: browsingContext.BrowsingContext / null,
}
</pre>

The <code>browsingContext.Info</code> type represents the properties of a
navigable.

<div algorithm>
To <dfn>get the child navigables</dfn> given |navigable|:

TODO: make this return a list in document order

1. Let |child navigables| be a [=/set=] containing all navigables that are
   a [=child navigable=] of |navigable|.

1. Return |child navigables|.

</div>


<div algorithm>
To <dfn>get the navigable info</dfn> given |navigable|,
|max depth| and |include parent id|:

1. Let |navigable id| be the [=navigable id=] for |navigable|.

1. Let |parent navigable| be |navigable|'s [=navigable/parent=].

1. If |parent navigable| is not null let |parent id| be the
   [=navigable id=] of |parent navigable|. Otherwise let
   |parent id| be null.

1. Let |document| be |navigable|'s [=active document=].

1. Let |url| be the result of running the [=URL serializer=], given
   |document|'s <a spec=dom>URL</a>.

   Note: This includes the fragment component of the URL.

1. Let |child infos| be null.

1. If |max depth| is null, or |max depth| is greater than 0:

  1. Let |child navigables| be [=get the child navigables=] given |navigable|.

  1. Let |child depth| be |max depth| - 1 if |max depth| is not null, or null otherwise.

  1. Set |child infos| to an empty [=/list=].

  1. For each |child navigable| of |child navigables|:

    1. Let |info| be the result of [=get the navigable info=] given
       |child navigable|, |child depth|, and false.

    1. Append |info| to |child infos|

1. Let |user context| be |navigable|'s [=associated user context=].

1. Let |opener id| be the [=navigable id=] for |navigable|'s
   [=original opener=], if |navigable|'s [=original opener=] is not null,
   and null otherwise.

1. Let |top-level traversable| be |navigable|'s  [=navigable/top-level
   traversable=].

1. Let |client window id| be the [=client window id=] for |top-level
   traversable|'s associated [=client window=].

1. Let |navigable info| be a [=/map=] matching the
   <code>browsingContext.Info</code> production with the <code>context</code>
   field set to |navigable id|, the <code>parent</code> field set to |parent id|
   if |include parent id| is <code>true</code>, or unset otherwise, the
   <code>url</code> field set to |url|, the <code>userContext</code> field set
   to |user context|'s [=user context id=], <code>originalOpener</code> field
   set to |opener id|, the <code>children</code> field set to |child infos|, and
   the <code>clientWindow</code> field set to |client window id|.

1. Return |navigable info|.

</div>

<div algorithm>
To <dfn>await a navigation</dfn> given |navigable|, |request|, |wait condition|, and optionally
|history handling| (default: "<code>default</code>") and |ignore cache| (default: false):

1. Let |navigation id| be the string representation of a
   [[!RFC9562|UUID]] based on truly random, or pseudo-random numbers.

1. [=Navigate=] |navigable| with resource |request|, and using
   |navigable|'s [=active document=] as the source
   {{Document}}, with [=navigation id=] |navigation id|, and [=history
   handling behavior=] |history handling|. If |ignore cache| is true,
   the navigation must not load resources from the HTTP cache.

   Issue: property specify how the |ignore cache| flag works. This needs to
   consider whether only the first load of a resource bypasses the cache
   (i.e. whether this is like initially clearing the cache and proceeding like
   normal), or whether resources not directly loaded by the HTML parser
   (e.g. loads initiated by scripts or stylesheets) also bypass the cache.

1.  Let (|event received|, |navigation status|) be [=await=] given
    «"<code>navigation started</code>", "<code>navigation failed</code>",
    "<code>fragment navigated</code>"», and |navigation id|.

1. Assert: |navigation status|'s [=WebDriver BiDi navigation status/id=] is |navigation id|.

1. If |navigation status|'s [=WebDriver BiDi navigation status/status=] is "<code>complete</code>":

  1. Let |body| be a [=/map=] matching the
     <code>browsingContext.NavigateResult</code> production, with the
     <code>navigation</code> field set to |navigation id|, and the
     <code>url</code> field set to the result of the [=URL serializer=] given
     |navigation status|'s [=WebDriver BiDi navigation status/url=].

  1. Return [=success=] with data |body|.

   Note: this is the case if the navigation only caused the fragment to
   change.

1. If |navigation status|'s [=WebDriver BiDi navigation status/status=] is "<code>canceled</code>"
   return [=error=] with [=error code=] [=unknown error=].

   TODO: is this the right way to handle errors here?

1. Assert: |navigation status|'s [=WebDriver BiDi navigation status/status=] is
   "<code>pending</code>" and |navigation id| is not null.

1. If |wait condition| is "<code>committed</code>", let |event name| be
   "<code>committed</code>".

1. Otherwise, if |wait condition| is "<code>interactive</code>", let |event name| be
   "<code>domContentLoaded</code>".

1. Otherwise, let |event name| be "<code>load</code>".

1. Let (|event received|, |status|) be [=await=] given «|event name|,
    "<code>download started</code>", "<code>navigation aborted</code>",
    "<code>navigation failed</code>"» and |navigation id|.

1. If |event received| is "<code>navigation failed</code>"
   return [=error=] with [=error code=] [=unknown error=].

   Issue: Are we surfacing enough information about what failed and why with
   an error here? What error code do we want? Is there going to be a problem
   where local ends parse the implementation-defined strings to figure out
   what actually went wrong?

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.NavigateResult</code> production, with the
   <code>navigation</code> field set to |status|'s id, and the
   <code>url</code> field set to the result of the [=URL serializer=] given
   |status|'s url.

1. Return [=success=] with data |body|.

</div>

#### The browsingContext.Locator Type #### {#type-browsingContext-Locator}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browsingContext.Locator = (
   browsingContext.AccessibilityLocator /
   browsingContext.CssLocator /
   browsingContext.ContextLocator /
   browsingContext.InnerTextLocator /
   browsingContext.XPathLocator
)

browsingContext.AccessibilityLocator = {
   type: "accessibility",
   value: {
    ? name: text,
    ? role: text,
   }
}

browsingContext.CssLocator = {
   type: "css",
   value: text
}

browsingContext.ContextLocator = {
  type: "context",
  value: {
    context: browsingContext.BrowsingContext,
  }
}

browsingContext.InnerTextLocator = {
   type: "innerText",
   value: text,
   ? ignoreCase: bool
   ? matchType: "full" / "partial",
   ? maxDepth: js-uint,
}

browsingContext.XPathLocator = {
   type: "xpath",
   value: text
}
</pre>

The <code>browsingContext.Locator</code> type provides details on the strategy
for locating a node in a document.

#### The browsingContext.Navigation Type #### {#type-browsingContext-Navigation}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browsingContext.Navigation = text;
</pre>

The <code>browsingContext.Navigation</code> type is a unique string identifying an ongoing
navigation.

TODO: Link to the definition in the HTML spec.

#### The browsingContext.Download Type #### {#type-browsingContext-Download}

{^remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browsingContext.Download = text;
</pre>

The <code>browsingContext.Download</code> type is a unique string identifying a download.


#### The browsingContext.NavigationInfo Type #### {#type-browsingContext-NavigationInfo}

{^local end definition^}:

<pre class="cddl" data-cddl-module="local-cddl">
browsingContext.BaseNavigationInfo = (
  context: browsingContext.BrowsingContext,
  navigation: browsingContext.Navigation / null,
  timestamp: js-uint,
  url: text,
  ? userContext: browser.UserContext,
)

browsingContext.NavigationInfo = {
  browsingContext.BaseNavigationInfo
}
</pre>

The <code>browsingContext.NavigationInfo</code> type provides details of an ongoing navigation.

<div algorithm>
To <dfn>get the navigation info</dfn>, given [=/navigable=] |navigable| and
[=WebDriver BiDi navigation status|navigation status=] |navigation status|:

1. Let |navigable id| be the [=navigable id=] for |navigable|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |timestamp| be a [=time value=] representing the current date and time in UTC.

1. Let |url| be |navigation status|'s [=WebDriver BiDi navigation status/url=].

1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

1. Return a [=/map=] matching the <code>browsingContext.NavigationInfo</code>
   production, with the <code>context</code> field set to |navigable id|, the
   <code>navigation</code> field set to |navigation id|, the
   <code>timestamp</code> field set to |timestamp|, the <code>url</code>
   field set to the result of the [=URL serializer=] given |url|, and the
   <code>userContext</code> field set to |user context id|.

</div>

#### The browsingContext.ReadinessState Type #### {#type-browsingContext-ReadinessState}

<pre class="cddl" data-cddl-module="remote-cddl">
browsingContext.ReadinessState = "none" / "interactive" / "complete"
</pre>

The <code>browsingContext.ReadinessState</code> type represents the stage of
document loading at which a navigation command will return.

#### The browsingContext.UserPromptType Type #### {#type-browsingContext-UserPromptType}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
browsingContext.UserPromptType = "alert" / "beforeunload" / "confirm" / "prompt";
</pre>

The <code>browsingContext.UserPromptType</code> type represents the possible user
prompt types.

### Commands ### {#module-browsingContext-commands}

#### The browsingContext.activate Command ####  {#command-browsingContext-activate}

The <dfn export for=commands>browsingContext.activate</dfn> command activates and focuses the given [=/top-level traversable=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Activate = (
        method: "browsingContext.activate",
        params: browsingContext.ActivateParameters
      )

      browsingContext.ActivateParameters = {
        context: browsingContext.BrowsingContext
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.ActivateResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.activate">

The [=remote end steps=] with |command parameters| are:

1. Let |navigable id| be the value of the |command parameters|["<code>context</code>"] field.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with |navigable id|.

1. If |navigable| is not a [=/top-level traversable=], return [=error=] with [=error code=] [=invalid argument=].

1. Return [=activate a navigable=] with |navigable|.

</div>

<div algorithm>
To <dfn>activate a navigable</dfn> given |navigable|:

1. Run implementation-specific steps so that |navigable|'s [=system visibility state=] becomes [=visible=]. If this is not possible return [=error=] with error code [=unsupported operation=].

    Note: This can have the side effect of making currently [=visible=] [=/navigables=] [=hidden=].

    Note: This can change the underlying OS state by causing the window to become unminimized or by other side effects related to changing the [=system visibility state=].

1. Run implementation-specific steps to set the [=top-level traversable/system focus=] on the |navigable| if it is not focused.

    Note: This does not change the [=focused area of the document=] except as mandated by other specifications.

1. Return [=success=] with data null.

</div>

#### The browsingContext.captureScreenshot Command ####  {#command-browsingContext-captureScreenshot}

The <dfn export for=commands>browsingContext.captureScreenshot</dfn> command
captures an image of the given navigable, and returns it as a
Base64-encoded string.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.CaptureScreenshot = (
        method: "browsingContext.captureScreenshot",
        params: browsingContext.CaptureScreenshotParameters
      )

      browsingContext.CaptureScreenshotParameters = {
        context: browsingContext.BrowsingContext,
        ? origin: ("viewport" / "document") .default "viewport",
        ? format: browsingContext.ImageFormat,
        ? clip: browsingContext.ClipRectangle,
      }

      browsingContext.ImageFormat = {
         type: text,
         ? quality: 0.0..1.0,
      }

      browsingContext.ClipRectangle = (
        browsingContext.BoxClipRectangle /
        browsingContext.ElementClipRectangle
      )

      browsingContext.ElementClipRectangle = {
        type: "element",
        element: script.SharedReference
      }

      browsingContext.BoxClipRectangle = {
         type: "box",
         x: float,
         y: float,
         width: float,
         height: float
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.CaptureScreenshotResult = {
          data: text
        }
    </pre>
   </dd>
</dl>

<div algorithm>
To <dfn>normalize rect</dfn> given |rect|:

Note: This ensures that the resulting rect has positive [=width dimension=] and
[=height dimension=].

1. Let |x| be |rect|'s [=x coordinate=].

1. Let |y| be |rect|'s [=y coordinate=].

1. Let |width| be |rect|'s [=width dimension=].

1. Let |height| be |rect|'s [=height dimension=].

1. If |width| is less than 0, set |x| to |x| + |width| and then set |width| to
   -|width|.

1. If |height| is less than 0, set |y| to |y| + |height| and then set |height|
   to -|height|.

1. Return a new {{DOMRectReadOnly}} with [=x coordinate=] |x|, [=y coordinate=]
   |y|, [=width dimension=] |width| and [=height dimension=] |height|.

</div>

<div algorithm>
To <dfn>rectangle intersection</dfn> given |rect1| and |rect2|

1. Let |rect1| be [=normalize rect=] with |rect1|.

1. Let |rect2| be [=normalize rect=] with |rect2|.

1. Let |x1_0| be |rect1|'s [=x coordinate=].

1. Let |x2_0| be |rect2|'s [=x coordinate=].

1. Let |x1_1| be |rect1|'s [=x coordinate=] plus |rect1|'s [=width dimension=].

1. Let |x2_1| be |rect2|'s [=x coordinate=] plus |rect2|'s [=width dimension=].

1. Let |x_0| be the maximum element of «|x1_0|, |x2_0|».

1. Let |x_1| be the minimum element of «|x1_1|, |x2_1|».

1. Let |y1_0| be |rect1|'s [=y coordinate=].

1. Let |y2_0| be |rect2|'s [=y coordinate=].

1. Let |y1_1| be |rect1|'s [=y coordinate=] plus |rect1|'s [=height dimension=].

1. Let |y2_1| be |rect2|'s [=y coordinate=] plus |rect2|'s [=height dimension=].

1. Let |y_0| be the maximum element of «|y1_0|, |y2_0|».

1. Let |y_1| be the minimum element of «|y1_1|, |y2_1|».

1. If |x_1| is less than |x_0|, let |width| be 0. Otherwise let |width| be
   |x_1| - |x_0|.

1. If |y_1| is less than |y_0|, let |height| be 0. Otherwise let |height| be
   |y_1| - |y_0|.

1. Return a new {{DOMRectReadOnly}} with [=x coordinate=] |x_0|, [=y coordinate=]
   |y_0|, [=width dimension=] |width| and [=height dimension=] |height|.

</div>

<div algorithm>
To <dfn>render document to a canvas</dfn> given |document| and |rect|:

1. Let |ratio| be [=determine the device pixel ratio=] given |document|'s
   [=default view=].

1. Let |paint width| be |rect|'s [=width dimension=] multiplied by |ratio|,
   rounded to the nearest integer, so it matches the width of |rect| in device
   pixels.

1. Let |paint height| be |rect|'s [=height dimension=] multiplied by |ratio|,
   rounded to the nearest integer, so it matches the height of |rect| in device
   pixels.

1. Let |canvas| be a new {{HTMLCanvasElement}} with {{HTMLCanvasElement/width}} |paint
   width| and {{HTMLCanvasElement/height}} |paint height|.

1. Let |canvas context| be the result of running the [=2D context creation algorithm=]
   with |canvas| and null.

1. Set |canvas|'s [=context mode=] to [=2D=].

1. Complete implementation specific steps equivalent to drawing the region of
   the framebuffer representing the region of |document| covered by |rect| to
   |canvas context|, such that each pixel in the framebuffer corresponds to a pixel in
   |canvas context| with (|rect|'s [=x coordinate=], |rect|'s [=y coordinate=]) in
   viewport coordinates corresponding to (0,0) in |canvas context| and (|rect|'s
   [=x coordinate=] + |rect|'s [=width dimension=], |rect|'s
   [=y coordinate=] + |rect|'s [=height dimension=]) corresponding to (|paint
   width|, |paint height|).

1. Return [=canvas=].

</div>

<div algorithm>
To <dfn>encode a canvas as Base64</dfn> given |canvas| and |format|:

1. If |format| is not null, let |type| be the <code>type</code> field of
   |format|, and let |quality| be the <code>quality</code> field of |format|.

1. Otherwise, let |type| be "image/png" and let |quality| be [=undefined=].

1. Let |file| be [=a serialization of the bitmap as a file=] for |canvas| with
   |type| and |quality|.

1. Let |encoded string| be the [=forgiving-base64 encode=] of |file|.

1. Return success with data |encoded string|.

</div>

<div algorithm>
To <dfn>get the origin rectangle</dfn> given |document| and |origin|:

1. If |origin| is <code>"viewport"</code>:

   1. Let |viewport| be |document|'s [=visual viewport=].

   1. Let |viewport rect| be a {{DOMRectReadOnly}} with [=x coordinate=]
      |viewport| [=visual viewport page left|page left=], [=y coordinate=]
      |viewport| [=visual viewport page top|page top=], [=width dimension=]
      |viewport| width, and [=height dimension=] |viewport| height.

   1. Return [=success=] with data |viewport rect|.

1. Assert: |origin| is <code>"document"</code>.

1. Let |document element| be the [=document element=] for |document|.

1. Let |document rect| be a {{DOMRectReadOnly}} with [=x coordinate=] 0, [=y
   coordinate=] 0, [=width dimension=] |document element| [=scroll height=],
   and [=height dimension=] |document element| [=scroll width=].

1. Return [=success=] with data |document rect|.

</div>

The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters| if present, or null otherwise.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
    with |navigable id|.

1. If the implementation is unable to capture a screenshot of |navigable| for any
   reason then return [=error=] with [=error code=] [=unsupported operation=].

1. Let |document| be |navigable|'s [=active document=].

1. Immediately after the next invocation of the [=run the animation frame
   callbacks=] algorithm for |document|:

   Issue(w3c/webdriver-bidi#1131): This ought to be integrated into the update
   rendering algorithm in some more explicit way.

1. Let |origin| be the value of the <code>context</code> field of |command
   parameters| if present, or "viewport" otherwise.

1. Let |origin rect| be the result of [=trying=] to [=get the origin rectangle=]
   given |origin| and |document|.

1. Let |clip rect| be |origin rect|.

1. If |command parameters| contains "<code>clip</code>":

  1. Let |clip| be |command parameters|["<code>clip</code>"].

  1. Run the steps under the first matching condition:
     <dl>
       <dt>|clip| matches the <code>browsingContext.ElementClipRectangle</code>
       production:
       <dd>
       1. Let |environment settings| be the [=environment settings object=] whose
          [=relevant global object=]'s <a>associated <code>Document</code></a> is
         |document|.

       1. Let |realm| be |environment settings|' [=realm execution context=]'s
          Realm component.

       1. Let |element| be the result of [=trying=] to [=deserialize remote reference=]
          with |clip|["<code>element</code>"], |realm|, and |session|.

       1. If |element| doesn't implement {{Element}} return [=error=] with [=error code=]
          [=no such element=].

       1. If |element|'s [=node document=] is not |document|, return [=error=] with
          [=error code=] [=no such element=].

       1. Let |viewport rect| be [=get the origin rectangle=] given
          "<code>viewport</code>" and |document|.

       1. Let |element rect| be [=get the bounding box=] for |element|.

       1. Let |clip rect| be a {{DOMRectReadOnly}} with [=x coordinate=]
          |element rect|["<code>x</code>"] + |viewport rect|["<code>x</code>"],
          [=y coordinate=] |element rect|["<code>y</code>"] + |viewport
          rect|["<code>y</code>"], width |element rect|["<code>width</code>"],
          and height |element rect|["<code>height</code>"].

       <dt>|clip| matches the <code>browsingContext.BoxClipRectangle</code> production:
       <dd>
       1. Let |clip x| be |clip|["<code>x</code>"] plus |origin rect|'s [=x
          coordinate=].

       1. Let |clip y| be |clip|["<code>y</code>"] plus |origin rect|'s [=y
          coordinate=].

       1. Let |clip rect| be a {{DOMRectReadOnly}} with [=x coordinate=] |clip x|,
          [=y coordinate=] |clip y|, width |clip|["<code>width</code>"], and height
          |clip|["<code>height</code>"].
     </dl>

1. Note: All coordinates are now measured from the origin of the document.

1. Let |rect| be the [=rectangle intersection=] of |origin rect| and |clip rect|.

1. If |rect|'s [=width dimension=] is 0 or |rect|'s [=height dimension=] is 0,
   return [=error=] with error code [=unable to capture screen=].

1. Let |canvas| be [=render document to a canvas=] with |document| and |rect|.

1. Let |format| be the <code>format</code> field of |command parameters|.

1. Let |encoding result| be the result of [=trying=] to [=encode a canvas as
   Base64=] with |canvas| and |format|.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.CaptureScreenshotResult</code> production, with the
   <code>data</code> field set to |encoding result|.

1. Return [=success=] with data |body|.

#### The browsingContext.close Command ####  {#command-browsingContext-close}

The <dfn export for=commands>browsingContext.close</dfn> command closes a
[=/top-level traversable=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Close = (
        method: "browsingContext.close",
        params: browsingContext.CloseParameters
      )

      browsingContext.CloseParameters = {
        context: browsingContext.BrowsingContext,
        ? promptUnload: bool .default false
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.CloseResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.close">
The [=remote end steps=] with |command parameters| are:

  1. Let |navigable id| be the value of the <code>context</code> field of
     |command parameters|.

  1. Let |prompt unload| be the value of the <code>promptUnload</code> field of
     |command parameters|.

  1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
     with |navigable id|.

  1. Assert: |navigable| is not null.

  1. If |navigable| is not a [=/top-level traversable=], return [=error=] with
     [=error code=] [=invalid argument=].

  1. If |prompt unload| is true:

     1. [=Close=] |navigable|.

  1. Otherwise:

     1. [=Close=] |navigable| without [=prompting to unload=].

  1. Return [=success=] with data null.

Issue(w3c/webdriver-bidi#170): There is an open discussion about the behavior
when closing the last [=/top-level traversable=]. We could expect to close
the browser, close the session or leave this up to the implementation.

</div>

#### The browsingContext.create Command ####  {#command-browsingContext-create}

The <dfn export for=commands>browsingContext.create</dfn> command creates a new
[=/navigable=], either in a new tab or in a new window, and returns its
[=navigable id=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Create = (
        method: "browsingContext.create",
        params: browsingContext.CreateParameters
      )

      browsingContext.CreateType = "tab" / "window"

      browsingContext.CreateParameters = {
        type: browsingContext.CreateType,
        ? referenceContext: browsingContext.BrowsingContext,
        ? background: bool .default false,
        ? userContext: browser.UserContext
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.CreateResult = {
          context: browsingContext.BrowsingContext,
          ? userContext: browser.UserContext
        }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.create">
The [=remote end steps=] with |command parameters| are:

  1. Let |type| be the value of the <code>type</code> field of
     |command parameters|.

  1. Let |reference navigable id| be the value of the <code>referenceContext</code>
     field of |command parameters|, if present, or null otherwise.

  1. If |reference navigable id| is not null, let |reference navigable| be the
     result of [=trying=] to [=get a navigable=] with
     |reference navigable id|. Otherwise let |reference navigable| be null.

  1. If |reference navigable| is not null and is not a [=/top-level traversable=],
    return [=error=] with [=error code=] [=invalid argument=].

  1. If the implementation is unable to create a new [=/top-level traversable=] for any
     reason then return [=error=] with [=error code=] [=unsupported operation=].

  1. Let |user context| be the [=default user context=] if |reference navigable|
     is null, and |reference navigable|' [=associated user context=] otherwise.

  1. Let |user context id| be the value of the <code>userContext</code> field of
     |command parameters| if present, or null otherwise.

  1. If |user context id| is not null, set |user context| to the result of
     [=trying=] to [=get user context=] with |user context id|.

  1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

  1. If the implementation is unable to create a new [=/top-level traversable=]
     with [=associated user context=] |user context| for any reason, return
     [=error=] with [=error code=] [=unsupported operation=].

  <!-- This is based on step 5 of https://w3c.github.io/webdriver/#new-window,
       but without using the "current browsing context" concept. -->
  1. Let |traversable| be the result of trying to [=/create a new top-level traversable=] steps with null and empty string,
     and setting the [=associated user context=] for the newly created [=/top-level traversable=] to |user context|.
     Which OS window the new [=/top-level traversable=] is created in depends on |type| and |reference navigable|:

     * If |type| is "<code>tab</code>" and the implementation supports
       multiple [=/top-level traversable=]s in the same OS window:

       * The new [=/top-level traversable=] should reuse an existing OS window, if any.

       * If |reference navigable| is not null, the new [=/top-level traversable=] should
         reuse the window containing |reference navigable|, if any. If the
         top-level traversables inside an OS window have a definite ordering,
         the new [=/top-level traversable=] should be immediately after
         |reference navigable|'s [=navigable/top-level traversable=] in that ordering.

     * If |type| is "<code>window</code>", and the implementation supports
       multiple [=/top-level traversable=] in separate OS windows, the created
       [=/top-level traversable=] should be in a new OS window.

     * Otherwise, the details of how the [=/top-level traversable=] is presented to the
       user are implementation defined.

  1. If the value of the |command parameters|' <code>background</code> field is false:

    1. Let |activate result| be the result of [=activate a navigable=] with the newly created [=/navigable=].

    1. If |activate result| is an [=error=], return |activate result|.

    Note: Do not invoke the [=/focusing steps=] for the created navigable if <code>background</code> is true.

  1. Let |body| be a [=/map=] matching the <code>browsingContext.CreateResult</code>
     production, with the <code>context</code> field set to |traversable|'s [=navigable id=] and
     the <code>userContext</code> property set to the [=user context id=] of |traversable|'s
     [=associated user context=].

  1. Return [=success=] with data |body|.

</div>

#### The browsingContext.getTree Command ####  {#command-browsingContext-getTree}

The <dfn export for=commands>browsingContext.getTree</dfn> command returns a
tree of all descendent navigables including the given parent itself,
or all top-level contexts when no parent is provided.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.GetTree = (
        method: "browsingContext.getTree",
        params: browsingContext.GetTreeParameters
      )

      browsingContext.GetTreeParameters = {
        ? maxDepth: js-uint,
        ? root: browsingContext.BrowsingContext,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.GetTreeResult = {
          contexts: browsingContext.InfoList
        }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.getTree">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |root id| be the value of the <code>root</code> field of
   |command parameters| if present, or null otherwise.

1. Let |max depth| be the value of the <code>maxDepth</code> field of |command
   parameters| if present, or null otherwise.

1. Let |navigables| be an empty [=/list=].

1. If |root id| is not null, append the result of [=trying=] to
   [=get a navigable=] given |root id| to |navigables|.
   Otherwise append all [=navigable/top-level traversables=] to |navigables|.

1. Let |navigables infos| be an empty [=/list=].

1. For each |navigable| of |navigables|:

  1. Let |info| be the result of [=get the navigable info=] given
     |navigable|, |max depth|, and true.

  1. Append |info| to |navigables infos|

1. Let |body| be a [=/map=] matching the <code>browsingContext.GetTreeResult</code>
   production, with the <code>contexts</code> field set to |navigables infos|.

1. Return [=success=] with data |body|.

</div>

#### The browsingContext.handleUserPrompt Command ####  {#command-browsingContext-handleUserPrompt}

The <dfn export for=commands>browsingContext.handleUserPrompt</dfn>
command allows closing an open prompt

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.HandleUserPrompt = (
        method: "browsingContext.handleUserPrompt",
        params: browsingContext.HandleUserPromptParameters
      )

      browsingContext.HandleUserPromptParameters = {
        context: browsingContext.BrowsingContext,
        ? accept: bool,
        ? userText: text,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.HandleUserPromptResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.handleUserPrompt">

The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Let |accept| be the value of the <code>accept</code> field of |command
   parameters| if present, or true otherwise.

1. Let |userText| be the value of the <code>userText</code> field of |command
   parameters| if present, or the empty string otherwise.

1. If |navigable| is currently showing a simple dialog from a call to [=alert=] then
   acknowledge the prompt.

   Otherwise if |navigable| is currently showing a simple dialog from a call to
   [=confirm=], then respond positively if |accept| is true, or respond
   negatively if |accept| is false.

   Otherwise if |navigable| is currently showing a simple dialog from a call to
   [=prompt=], then respond with the string value |userText| if |accept| is
   true, or abort if |accept| is false.

   Otherwise, if |navigable| is currently showing a prompt as part of the [=prompt
   to unload=] steps, then confirm the navigation if |accept| is true, otherwise
   refuse the navigation.

   Otherwise return [=error=] with [=error code=] [=no such alert=].

1. Return [=success=] with data null.

</div>
#### The browsingContext.locateNodes Command ####  {#command-browsingContext-locateNodes}

The <dfn export for=commands>browsingContext.locateNodes</dfn> command returns a
list of all nodes matching the specified locator.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.LocateNodes = (
        method: "browsingContext.locateNodes",
        params: browsingContext.LocateNodesParameters
      )

      browsingContext.LocateNodesParameters = {
         context: browsingContext.BrowsingContext,
         locator: browsingContext.Locator,
         ? maxNodeCount: (js-uint .ge 1),
         ? serializationOptions: script.SerializationOptions,
         ? startNodes: [ + script.SharedReference ]
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.LocateNodesResult = {
            nodes: [ * script.NodeRemoteValue ]
        }
    </pre>
   </dd>
</dl>

<div algorithm="locate nodes using CSS">
To <dfn>locate nodes using CSS</dfn> with given |navigable|, |context nodes|,
|selector|, |maximum returned node count|, and <var ignore>session</var>:

1. Let |returned nodes| be an empty [=/list=].

1. Let |parse result| be the result of [=parse a selector=] given |selector|.

1. If |parse result| is failure, return [=error=] with [=error code=] [=invalid selector=].

1. For each |context node| of |context nodes|:

  1. Let |elements| be the result of [=match a selector against a tree=] with
     |parse result| and |navigable|’s [=active document=] [=root=] using
     [=scoping root=] |context node|.

  1. For each |element| in |elements|:

    1. [=list/Append=] |element| to |returned nodes|.

    1. If |maximum returned node count| is not null and [=list/size=] of
       |returned nodes| is equal to |maximum returned node count|,
       return [=success=] with data |returned nodes|.

1. Return [=success=] with data |returned nodes|.

</div>

<div algorithm="locate the container element">
To <dfn>locate the container element</dfn> given |navigable|:

1. Let |returned nodes| be an empty [=/list=].

1. If |navigable|'s [=navigable/container=] is not null,
   append |navigable|'s [=navigable/container=] to |returned nodes|.

1. Return |returned nodes|.

</div>

<div algorithm="locate nodes using XPath">

To <dfn>locate nodes using XPath</dfn> with given |navigable|, |context nodes|,
|selector|, and |maximum returned node count|:

Note: Owing to the unmaintained state of the XPath specification, this algorithm
is phrased as if making calls to the XPath DOM APIs. However this is to be understood
as equivalent to spec-internal calls directly accessing the underlying algorithms,
without going via the ECMAScript runtime.

1. Let |returned nodes| be an empty [=/list=].

1. For each |context node| of |context nodes|:

   1. Let |evaluate result| be the result of calling [=evaluate=] on |navigable|'s
      [=active document=], with arguments |selector|, |context node|, null,
      [=ORDERED_NODE_SNAPSHOT_TYPE=], and null. If this throws a "[=SyntaxError=]"
      [=DOMException=], return [=error=] with [=error code=] [=invalid selector=];
      otherwise, if this throws any other exception return [=error=] with [=error code=]
      [=unknown error=].

   1. Let |index| be 0.

   1. Let |length| be the result of getting the <code>snapshotLength</code> property
      from |evaluate result|.

   1. Repeat, while |index| is less than |length|:

      1. Let |node| be the result of calling [=snapshotItem=] with |evaluate result|
         as this and |index| as the argument.

      1. [=list/Append=] |node| to |returned nodes|.

      1. If |maximum returned node count| not null and [=list/size=] of
         |returned nodes| is equal to |maximum returned node count|,
         return [=success=] with data |returned nodes|.

      1. Set |index| to |index| + 1.

1. Return [=success=] with data |returned nodes|.

</div>

<div algorithm="locate nodes using inner text">
To <dfn>locate nodes using inner text</dfn> with given |context nodes|,
|selector|, |max depth|, |match type|, |ignore case|, and |maximum returned node count|:

1. If |selector| is the empty string, return [=error=] with [=error code=]
   [=invalid selector=].

1. Let |returned nodes| be an empty [=/list=].

1. If |ignore case| is false, let |search text| be |selector|. Otherwise, let
   |search text| be the result of [=toUppercase=] with |selector| according
   to the [=Unicode Default Case Conversion algorithm=].

1. For each |context node| in |context nodes|:

  1. If |context node| implements {{Document}} or {{DocumentFragment}}:

    Note: when traversing the document or document fragment, <code>max depth</code>
    is not decreased intentionally to make the search result with
    <code>document</code> and <code>document.documentElement</code> equivalent.

    1. Let |child nodes| be an empty [=/list=].

    1. For each node |child| in the <a spec=dom>children</a> of |context node|.

      1. [=list/Append=] |child| to |child nodes|.

    1. [=list/Extend=] |returned nodes| with the result of [=trying=] to [=locate nodes using inner text=]
      with |child nodes|, |selector|, |max depth|, |match type|, |ignore case|, and |maximum returned node count|.

  1. If |context node| does not implement {{HTMLElement}} then [=continue=].

  1. Let |node inner text| be the result of calling the [=innerText getter steps=] with
     |context node| as the [=this=] value.

  1. If |ignore case| is false, let |node text| be |node inner text|. Otherwise,
     let |node text| be the result of [=toUppercase=] with |node inner text|
     according to the [=Unicode Default Case Conversion algorithm=].

  1. If |search text| is a [=code point substring=] of |node text|, perform the
     following steps:

    1. Let |child nodes| be an empty [=/list=] and, for each node |child| in the
       <a spec=dom>children</a> of |context node|:

      1. [=list/Append=] |child| to |child nodes|.

    1. If [=list/size=] of |child nodes| is equal to 0 or |max depth| is equal to 0,
       perform the following steps:

      1. If |match type| is <code>"full"</code> and |node text| [=is=] |search text|,
         [=list/append=] |context node| to |returned nodes|.

      1. Otherwise, if |match type| is <code>"partial"</code>, [=list/append=]
         |context node| to |returned nodes|.

    1. Otherwise, perform the following steps:

      1. Let |child max depth| be null if |max depth| is null, or |max depth| - 1 otherwise.

      1. Let |child node matches| be the result of [=locate nodes using inner text=]
         with |child nodes|, |selector|, |child max depth| , |match type|,
         |ignore case|, and |maximum returned node count|.

      1. If [=list/size=] of |child node matches| is equal to 0 and |match type| is
         <code>"partial"</code>, append |context node| to |returned nodes|. Otherwise,
         [=list/extend=] |returned nodes| with |child node matches|.

1. If |maximum returned node count| is not null, [=list/remove=] all entries
   in |returned nodes| with an index greater than or equal to |maximum returned
   node count|.

1. Return [=success=] with data |returned nodes|.

</div>

<div algorithm="collect nodes using accessibility attributes">
To <dfn>collect nodes using accessibility attributes</dfn> with given |context nodes|, |selector|,
|maximum returned node count|, and |returned nodes|:

1. If |returned nodes| is null:

  1. Set |returned nodes| to an empty [=/list=].

1. For each |context node| in |context nodes|:

  1. Let |match| be true.

  1. If |context node| implements {{Element}}:

    1. If |selector| [=map/contains=] "<code>role</code>":

      1. Let |role| be the [=computed role=] of |context node|.

      1. If |selector|["<code>role</code>"] [=is|is not=] |role|:

        1. Set |match| to false.

    1. If |selector| [=map/contains=] "<code>name</code>":

      1. Let |name| be the [=accessible name=] of |context node|.

      1. If |selector|["<code>name</code>"] [=is|is not=] |name|:

        1. Set |match| to false.

  1. Otherwise, set |match| to false.

  1. If |match| is true:

    1. If |maximum returned node count| is not null and [=list/size=] of |returned
      nodes| is equal to |maximum returned node count|, [=break=].

    1. [=list/Append=] |context node| to |returned nodes|.

  1. Let |child nodes| be an empty [=/list=] and, for each node |child| in the
     <a spec=dom>children</a> of |context node|:

    1. If |child| implements {{Element}}, [=list/append=] |child| to |child nodes|.

  1. [=Try=] to [=collect nodes using accessibility attributes=] with |child nodes|,
     |selector|, |maximum returned node count|, and |returned nodes|.

1. Return |returned nodes|.

</div>

<div algorithm="locate nodes using accessibility attributes">
To <dfn>locate nodes using accessibility attributes</dfn> with given |context nodes|, |selector|, and
|maximum returned node count|:

1. If |selector| does not [=map/contain=] "<code>role</code>" and
   |selector| does not [=map/contain=] "<code>name</code>", return
   [=error=] with [=error code=]
   [=invalid selector=].

1. Return the result of [=collect nodes using accessibility attributes=] with |context nodes|,
    |selector|, |maximum returned node count|, and null.

</div>

<div algorithm="remote end steps for browsingContext.locateNodes">
The [=remote end steps=] with |session| and |command parameters| are:

1. Let |navigable id| be |command parameters|["<code>context</code>"].

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Assert: |navigable| is not null.

1. Let |realm| be the result of [=trying=] to [=get a realm from a navigable=]
   with [=navigable id=] of |navigable| and null.

1. Let |locator| be |command parameters|["<code>locator</code>"].

1. If |command parameters| [=map/contains=] "<code>startNodes</code>", let
   |start nodes parameter| be |command parameters|["<code>startNodes</code>"].
   Otherwise let |start nodes parameter| be null.

1. If |command parameters| [=map/contains=] "<code>maxNodeCount</code>", let
   |maximum returned node count| be |command parameters|["<code>maxNodeCount</code>"].
   Otherwise, let |maximum returned node count| be null.

1. Let |context nodes| be an empty [=/list=].

1. If |start nodes parameter| is null, [=list/append=] the |navigable|'s
   [=active document=] to |context nodes|. Otherwise, for each
   |serialized start node| in |start nodes parameter|:

   1. Let |start node| be the result of [=trying=] to [=deserialize shared reference=] given
      |serialized start node|, |realm| and |session|.

   1. [=list/Append=] |start node| to |context nodes|.

1. Assert [=list/size=] of |context nodes| is greater than 0.

1. Let |type| be |locator|["<code>type</code>"].

1. In the following list of conditions and associated steps, run the first set
   of steps for which the associated condition is true:

   <dl>

      <dt>|type| is the string "<code>css</code>"
      <dd>
         1. Let |selector| be |locator|["<code>value</code>"].

         1. Let |result nodes| be a result of [=trying=] to [=locate nodes using css=]
            given |navigable|, |context nodes|, |selector| and |maximum returned
            nodes|.

      <dt>|type| is the string "<code>xpath</code>"
      <dd>
         1. Let |selector| be |locator|["<code>value</code>"].

         1. Let |result nodes| be a result of [=trying=] to [=locate nodes using xpath=]
            given |navigable|, |context nodes|, |selector| and |maximum returned
            nodes|.

      <dt>|type| is the string "<code>innerText</code>"
      <dd>
         1. Let |selector| be |locator|["<code>value</code>"].

         1. If |locator| [=map/contains=] <code>maxDepth</code>, let |max depth| be
            |locator|["<code>maxDepth</code>"]. Otherwise, let |max depth| be null.

         1. If |locator| [=map/contains=] <code>ignoreCase</code>, let |ignore case| be
            |locator|["<code>ignoreCase</code>"]. Otherwise, let |ignore case| be false.

         1. If |locator| [=map/contains=] <code>matchType</code>, let |match type| be
            |locator|["<code>matchType</code>"]. Otherwise, let |match type| be "full".

         1. Let |result nodes| be a result of [=trying=] to [=locate nodes using inner text=]
            given |context nodes|, |selector|, |max depth|, |match type|,
            |ignore case| and |maximum returned node count|.

      <dt>|type| is the string "<code>accessibility</code>"
      <dd>
         1. Let |selector| be |locator|["<code>value</code>"].

         1. Let |result nodes| be [=locate nodes using accessibility attributes=]
            given |context nodes|, |selector|, and |maximum returned node count|.

      <dt>|type| is the string "<code>context</code>"
      <dd>

         1. If |start nodes parameter| is not null,
            return [=error=] with [=error code=] "<code>invalid argument</code>".

         1. Let |selector| be |locator|["<code>value</code>"].

         1. Let |context id| be |selector|["<code>context</code>"].

         1. Let |child navigable| be the result of [=trying=] to [=get a navigable=] with |context id|.

         1. If |child navigable|'s [=navigable/parent=] is not |navigable|,
            return [=error=] with [=error code=] "<code>invalid argument</code>".

         1. Let |result nodes| be [=locate the container element=] given |child navigable|.

         1. Assert: For each |node| in |result nodes|, |node|'s [=/node navigable=] is |navigable|.

    </dl>

1. Assert: |maximum returned node count| is null or [=list/size=] of |result nodes| is less
   than or equal to |maximum returned node count|.

1. If |command parameters| [=map/contains=] "<code>serializationOptions</code>",
   let |serialization options| be |command parameters|["<code>serializationOptions</code>"].
   Otherwise, let |serialization options| be a [=/map=] matching the
   <code>script.SerializationOptions</code> production with the fields
   set to their default values.

1. Let |result ownership| be "none".

1. Let |serialized nodes| be an empty [=/list=].

1. For each |result node| in |result nodes|:

   1. Let |serialized node| be the result of [=serialize as a remote value=] with
      |result node|, |serialization options|, |result ownership|, a new [=/map=]
      as serialization internal map, |realm| and |session|.

   1. [=list/Append=] |serialized node| to |serialized nodes|.

1. Let |result| be a [=/map=] matching the <code>browsingContext.LocateNodesResult</code>
   production, with the <code>nodes</code> field set |serialized nodes|.

1. Return [=success=] with data |result|.

</div>

#### The browsingContext.navigate Command ####  {#command-browsingContext-navigate}

The <dfn export for=commands>browsingContext.navigate</dfn> command navigates a
navigable to the given URL.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Navigate = (
        method: "browsingContext.navigate",
        params: browsingContext.NavigateParameters
      )

      browsingContext.NavigateParameters = {
        context: browsingContext.BrowsingContext,
        url: text,
        ? wait: browsingContext.ReadinessState,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.NavigateResult = {
          navigation: browsingContext.Navigation / null,
          url: text,
        }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.navigate">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Assert: |navigable| is not null.

1. Let |wait condition| be "<code>committed</code>".

1. If |command parameters| [=map/contains=] <code>wait</code>
   and |command parameters|[<code>wait</code>] is not "<code>none</code>",
   set |wait condition| to |command parameters|[<code>wait</code>].

1. Let |url| be the value of the <code>url</code> field of |command
   parameters|.

1. Let |document| be |navigable|'s [=active document=].

1. Let |base| be |document|'s [=base URL=].

1. Let |url record| be the result of applying the [=URL parser=] to |url|,
   with [=base URL=] |base|.

1. If |url record| is failure, return [=error=] with [=error code=] [=invalid
   argument=].

1. Let |request| be a new [=/request=] whose URL is |url record|.

1. Return the result of [=await a navigation=] with |navigable|, |request| and
   |wait condition|.

</div>

#### The browsingContext.print Command ####  {#command-browsingContext-print}

The <dfn export for=commands>browsingContext.print</dfn> command
creates a paginated representation of a document, and returns it as a
PDF document represented as a Base64-encoded string.


<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Print = (
        method: "browsingContext.print",
        params: browsingContext.PrintParameters
      )

      browsingContext.PrintParameters = {
        context: browsingContext.BrowsingContext,
        ? background: bool .default false,
        ? margin: browsingContext.PrintMarginParameters,
        ? orientation: ("portrait" / "landscape") .default "portrait",
        ? page: browsingContext.PrintPageParameters,
        ? pageRanges: [*(js-uint / text)],
        ? scale: (0.1..2.0) .default 1.0,
        ? shrinkToFit: bool .default true,
      }

      browsingContext.PrintMarginParameters = {
        ? bottom: (float .ge 0.0) .default 1.0,
        ? left: (float .ge 0.0) .default 1.0,
        ? right: (float .ge 0.0) .default 1.0,
        ? top: (float .ge 0.0) .default 1.0,
      }

      ; Minimum size is 1pt x 1pt. Conversion follows from
      ; https://www.w3.org/TR/css3-values/#absolute-lengths
      browsingContext.PrintPageParameters = {
        ? height: (float .ge 0.0352) .default 27.94,
        ? width: (float .ge 0.0352) .default 21.59,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.PrintResult = {
          data: text
        }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.print">

The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. If the implementation is unable to provide a paginated representation of
   |navigable| for any reason then return [=error=] with [=error code=]
   [=unsupported operation=].

1. Let |margin| be the value of the <code>margin</code> field of |command
   parameters| if present, or otherwise a [=/map=] matching the
   <code>browsingContext.PrintMarginParameters</code> with the fields set to
   their default values.

1. Let |page size| be the value of the <code>page</code> field of |command
   parameters| if present, or otherwise a [=/map=] matching the
   <code>browsingContext.PrintPageParameters</code> with the fields set to
   their default values.

Note: The minimum page size is 1 point, which is (2.54 / 72) cm as per
[=absolute lengths=].

1. Let |page ranges| be the value of the <code>pageRanges</code> field of
   |command parameters| if present or an empty [=/list=] otherwise.

1. Let |document| be |navigable|'s [=active document=].

1. Immediately after the next invocation of the [=run the animation frame
   callbacks=] algorithm for |document|:

   Issue(w3c/webdriver-bidi#1131): This ought to be integrated into the update
   rendering algorithm in some more explicit way.

  1. Let |pdf data| be the result taking UA-specific steps to generate a
     paginated representation of |document|, with the CSS [=media type=] set to
     <code>print</code>, encoded as a PDF, with the following paper settings:

     <table>
       <tr>
         <th>Property
         <th>Value
       <tr>
         <td>Width in cm
         <td>|page size|["<code>width</code>"] if
             |command parameters|["<code>orientation</code>"] is
             "<code>portrait</code>" otherwise |page size|["<code>height</code>"]
       <tr>
         <td>Height in cm
         <td>|page size|["<code>height</code>"] if
             |command parameters|["<code>orientation</code>"] is
             "<code>portrait</code>" otherwise |page size|["<code>width</code>"]
       <tr>
         <td>Top margin, in cm
         <td>|margin|["<code>top</code>"]
       <tr>
         <td>Bottom margin, in cm
         <td>|margin|["<code>bottom</code>"]
       <tr>
         <td>Left margin, in cm
         <td>|margin|["<code>left</code>"]
       <tr>
         <td>Right margin, in cm
         <td>|margin|["<code>right</code>"]
     </table>

     In addition, the following formatting hints should be applied by the UA:
       <dl>
         <dt>If |command parameters|["<code>scale</code>"] is not equal to <code>1</code>:
         <dd>Zoom the size of the content by a factor |command parameters|["<code>scale</code>"]
         <dt>If |command parameters|["<code>background</code>"] is false:
         <dd>Suppress output of background images
         <dt>If |command parameters|["<code>shrinkToFit</code>"] is true:
         <dd>Resize the content to match the page width, overriding any page
             width specified in the content
       </dl>

  1. If |page ranges| is not [=list/empty=], let |pages| be the result of
     [=trying=] to [=parse a page range=] with |page ranges| and the number of
     pages contained in |pdf data|, then remove any pages from |pdf data|
     whose one-based index is not contained in |pages|.

  1. Let |encoding result| be the result of calling [=Base64 Encode=] on
     |pdf data|.

  1. Let |encoded data| be |encoding result|’s data.

  1. Let |body| be a [=/map=] matching the
     <code>browsingContext.PrintResult</code> production, with the
     <code>data</code> field set to |encoded data|.

  1. Return [=success=] with data |body|.

</div>

#### The browsingContext.reload Command ####  {#command-browsingContext-reload}

The <dfn export for=commands>browsingContext.reload</dfn> command reloads a
navigable.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.Reload = (
        method: "browsingContext.reload",
        params: browsingContext.ReloadParameters
      )

      browsingContext.ReloadParameters = {
        context: browsingContext.BrowsingContext,
        ? ignoreCache: bool,
        ? wait: browsingContext.ReadinessState,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.ReloadResult = browsingContext.NavigateResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.reload">
The [=remote end steps=] with |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Assert: |navigable| is not null.

1. Let |ignore cache| be the the value of the <code>ignoreCache</code> field of |command
   parameters| if present, or false otherwise.

1. Let |wait condition| be "<code>committed</code>".

1. If |command parameters| [=map/contains=] <code>wait</code>
   and |command parameters|[<code>wait</code>] is not "<code>none</code>",
   set |wait condition| to |command parameters|[<code>wait</code>].

1. Let |document| be |navigable|'s [=active document=].

1. Let |url| be |document|'s <a spec=DOM>URL</a>.

1. Let |request| be a new [=/request=] whose URL is |url|.

1. Return the result of [=await a navigation=] with |navigable|, |request|, |wait
   condition|, history handling "<code>reload</code>", and ignore
   cache |ignore cache|.

</div>

#### The browsingContext.setBypassCSP Command ####  {#command-browsingContext-setBypassCSP}

The <dfn export for=commands>browsingContext.setBypassCSP</dfn> command allows bypassing Content Security Policy enforcement.

Note: When CSP bypass is enabled, all CSP directives are bypassed, including those that would normally block eval(), new Function(), inline scripts, and resource loading.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.SetBypassCSP = (
        method: "browsingContext.setBypassCSP",
        params: browsingContext.SetBypassCSPParameters
      )

      browsingContext.SetBypassCSPParameters = {
        bypass: true / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.SetBypassCSPResult = EmptyResult
      </pre>
   </dd>
</dl>

A [=remote end=] has a <dfn>bypass CSP configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] boolean.

<div algorithm>

The <dfn export>WebDriver BiDi CSP is bypassed</dfn> steps given
[=/navigable=] |navigable| are:

1. Let |top-level traversable| be |navigable|'s [=navigable/top-level traversable=].

1. Let |bypass CSP enabled| be the result of [=get WebDriver configuration value=] of
   [=bypass CSP configuration=] for |top-level traversable|.

1. Assert: |bypass CSP enabled| is <code>true</code> or [=WebDriver configuration/unset=].

1. If |bypass CSP enabled| is [=WebDriver configuration/unset=], return false.

1. Return true.

</div>

<div algorithm="remote end steps for browsingContext.setBypassCSP">

The [=remote end steps=] given |command parameters| are:

1. Let |bypass| be |command parameters|["<code>bypass</code>"].

1. If |bypass| is null, set |bypass| to [=WebDriver configuration/unset=].

1. [=Try=] to [=store WebDriver configuration=] [=bypass CSP configuration=] |bypass| for |command parameters|.

1. Return [=success=] with data null.

</div>

#### The browsingContext.setViewport Command ####  {#command-browsingContext-setViewport}

The <dfn export for=commands>browsingContext.setViewport</dfn> command modifies specific viewport characteristics (e.g. viewport width and viewport height) on the given top-level traversable.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.SetViewport = (
        method: "browsingContext.setViewport",
        params: browsingContext.SetViewportParameters
      )

      browsingContext.SetViewportParameters = {
        ? context: browsingContext.BrowsingContext,
        ? viewport: browsingContext.Viewport / null,
        ? devicePixelRatio: (float .gt 0.0) / null,
        ? userContexts: [+browser.UserContext],
      }

      browsingContext.Viewport = {
        width: js-uint,
        height: js-uint,
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.SetViewportResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
To <dfn>set device pixel ratio override</dfn> given |navigable| and |device pixel ratio|:

1. If |device pixel ratio| is not null:

   1. For [=/document=] currently loaded in a specified |navigable|:

      1. When the [=select an image source from a source set=] steps are run, act as if
         the implementation's pixel density was set to |device pixel ratio| when selecting an image.

      1. For the purposes of the [=resolution media feature=], act as if
         the implementation's resolution is |device pixel ratio| dppx scaled by the page zoom.

   1. [=map/Set=] [=device pixel ratio overrides=][|navigable|] to |device pixel ratio|.

      Note: This will take an effect because of the patch of [[#patchs-determine-the-device-pixel-ratio]].

1. Otherwise:

   1. For [=/document=] currently loaded in a specified |navigable|:

      1. When the [=select an image source from a source set=] steps are run, use the implementation's default behavior,
         without any changes made by previous invocations of these steps.

      1. For the purposes of the [=resolution media feature=], use the implementation's default behavior,
         without any changes made by previous invocations of these steps.

   1. [=map/Remove=] |navigable| from [=device pixel ratio overrides=].

1. Run [=evaluate media queries and report changes=] for [=/document=] currently loaded
   in a specified |navigable|.

</div>


<div algorithm>
To <dfn>set viewport</dfn> given given [=/navigable=] |navigable| and
[=viewport-configuration/viewport=] |viewport|:

1. If |viewport| is not null, set the width of |navigable|'s [=layout viewport=]
   to be the |viewport|'s [=viewport-dimensions/width=] in CSS pixels and
   set the height of the |navigable|'s [=layout viewport=] to be the
   |viewport|'s [=viewport-dimensions/height=] in CSS pixels.

1. Otherwise, set the |navigable|'s [=layout viewport=] to the
   implementation-defined default.

</div>

<div algorithm="configure new navigable">

After creating a document in a new [=/navigable=] |navigable| and
before the [=run WebDriver BiDi preload scripts=] algorithm is invoked:

TODO: Move it as a hook in the html spec instead.

1. Let |user context| be |navigable|'s [=associated user context=].

1. If |navigable| is a [=/top-level traversable=]:

   1. [=Update geolocation override=] for |navigable|.

   1. [=Update emulated forced colors theme=] for |navigable|.

   1. If [=screen orientation overrides map=] [=map/contains=] |user context|,
      [=set emulated screen orientation=] with |navigable| and
      [=screen orientation overrides map=][|user context|].

1. If [=viewport overrides map=] [=map/contains=] |user context|:

   1. If |navigable| is a [=/top-level traversable=] and [=viewport overrides map=][|user context|]'s
      [=viewport-configuration/viewport=] is not null:

      1. [=Set viewport=] with |navigable| and [=viewport overrides map=][|user context|]'s
         [=viewport-configuration/viewport=].

   1. If [=viewport overrides map=][|user context|]'s [=viewport-configuration/devicePixelRatio=] is not null:

      1. [=Set device pixel ratio override=] with |navigable| and [=viewport overrides map=][|user context|]'s
         [=viewport-configuration/devicePixelRatio=].

1. [=Update scrollbar type override=] for |navigable|.

</div>

<div algorithm="remote end steps for browsingContext.setViewport">

The [=remote end steps=] with |command parameters| are:

1. If the implementation is unable to adjust the [=layout viewport=]
   parameters with the given |command parameters| for any reason, return
   [=error=] with [=error code=] [=unsupported operation=].

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>context</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |navigables| be a [=/set=].

1. If the <code>context</code> field of |command parameters| is present:

   1. Let |navigable id| be the value of the <code>context</code> field of |command
      parameters|.

   1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with
      |navigable id|.

   1. If |navigable| is not a [=/top-level traversable=], return [=error=] with
      [=error code=] [=invalid argument=].

   1. [=set/Append=] |navigable| to |navigables|.

1. Otherwise, if the <code>userContexts</code> field of |command parameters| is present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=] with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. [=map/Set=] [=viewport overrides map=][|user context|] to a struct.

      1. If |command parameters| [=map/contains=] "<code>viewport</code>":

         1. Set [=viewport overrides map=][|user context|]'s [=viewport-configuration/viewport=]
            to |command parameters|["<code>viewport</code>"].

      1. If |command parameters| [=map/contains=] "<code>devicePixelRatio</code>":

         1. Set [=viewport overrides map=][|user context|]'s [=viewport-configuration/devicePixelRatio=]
            to |command parameters|["<code>devicePixelRatio</code>"].

      1. [=list/For each=] |top-level traversable| of the list of all [=/top-level traversables=]
         whose [=associated user context=] is |user context|:

         1. [=list/Append=] |top-level traversable| to |navigables|.

1. Otherwise, return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| [=map/contains=] the <code>viewport</code> field:

   1. Let |viewport| be the |command parameters|["<code>viewport</code>"].

   1. For each |navigable| of |navigables|:

      1. [=Set viewport=] with |navigable| and |viewport|.

      1. Run the [[cssom-view-1#resizing-viewports]] steps with |navigable|'s [=active document=].

1. If |command parameters| [=map/contains=] the <code>devicePixelRatio</code> field:

   1. Let |device pixel ratio| be the |command
      parameters|["<code>devicePixelRatio</code>"].

   1. For each |navigable| of |navigables|:

      1. For the |navigable| and all [=descendant navigables=]:

         1. [=Set device pixel ratio override=] with |navigable| and |device pixel ratio|.

1. Return [=success=] with data null.

</div>

#### The browsingContext.startScreencast Command ####  {#command-browsingContext-startScreencast}

The <dfn export for=commands>browsingContext.startScreencast</dfn> command
starts the screencast of a given navigable and writes it to a file.

Note: The [=remote end=] creates and writes the screencast file, but does not delete it.
Cleaning up the file is left to the [=local end=]. In some configurations this might not be
possible — for example, if the [=remote end=] has read/write access to the filesystem but
the [=local end=] has only read-only access.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.StartScreencast = (
        method: "browsingContext.startScreencast",
        params: browsingContext.StartScreencastParameters
      )

      browsingContext.StartScreencastParameters = {
        context: browsingContext.BrowsingContext,
        ? mimeType: text,
        ? video: browsingContext.MediaTrackConstraints,
        ? audio: bool .default false,
      }

      browsingContext.MediaTrackConstraints = {
         ? width: js-uint,
         ? height: js-uint,
         ? frameRate: js-uint,
      }

      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.StartScreencastResult = {
         screencast: browsingContext.Screencast,
         path: text
      }

    </pre>
    <pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
      browsingContext.Screencast = text
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.startScreencast">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |navigable id| be |command parameters|["<code>context</code>"].

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. If |navigable| is not a [=/top-level traversable=], return [=error=] with
   [=error code=] [=invalid argument=].

1. If |command parameters| [=map/contains=] the <code>mimeType</code> field:

   1. Let |mime type| be |command parameters|["<code>mimeType</code>"].

1. Otherwise, set |mime type| to the implementation-defined default format.

1. If the implementation is unable to record a screencast of |navigable| for any
   reason then return [=error=] with [=error code=] [=unsupported operation=].

1. Let |stream| be a new [=screencast stream=] for |navigable|, constructed as follows:

   1. Create a |video track| which must be a live-capture of the [=/browser=] [=display surface=]
      of the [=relevant global object=]'s [=associated `Document`=]'s |navigable|'s
      <a spec=css2>viewport</a>.

   1. If |command parameters| [=map/contains=] "<code>video</code>":

      1. Let |video| be |command parameters|["<code>video</code>"].

      1. If |video| [=map/contains=] <code>width</code>:

         1. The user agent must attempt to obtain media whose width is as close as possible
            to |video|["<code>width</code>"] given the capabilities
            of the hardware and the other constraints specified.

      1. If |video| [=map/contains=] <code>height</code>:

         1. The user agent must attempt to obtain media whose height is as close as possible
            to |video|["<code>height</code>"] given the capabilities
            of the hardware and the other constraints specified.

      1. If |video| [=map/contains=] <code>frameRate</code>:

         1. The user agent must attempt to obtain media whose frame rate is as close as possible
            to |video|["<code>frameRate</code>"] given the capabilities
            of the hardware and the other constraints specified.

   1. [=map/Set=] |video track| to |stream|[[=screencast stream/video track=]].

   1. If |command parameters|["<code>audio</code>"] is true:

      1. Create an |audio track| which contains the combined audio produced by the sum of documents
         that consist of the [=relevant global object=]'s [=associated `Document`=]'s
         |navigable|'s [=navigable/active document=], and all [=navigable/active documents=]
         in nested [=/browsing context=]s of the [=relevant global object=]'s [=associated `Document`=]'s |navigable|.

      1. [=map/Set=] |audio track| to |stream|[[=screencast stream/audio track=]].

   1. If the implementation is unable to produce |stream|, return [=error=] with
      [=error code=] [=unknown error=].

1. Let |path| be an implementation-defined file path where the recording will
   be stored.

1. Let |screencast| be the string representation of a [[!RFC9562|UUID]].

1. Let |recording| be a new [=screencast recording=] with
   [=screencast recording/stream=] |stream|,
   [=screencast recording/path=] |path|,
   [=screencast recording/state=] "<code>recording</code>",
   [=screencast recording/writeError=] null.

1. Set |session|'s [=screencast recordings map=][|screencast|] to |recording|.

1. [=Start a screencast recording=] given |recording| and |mime type|.

1. Let |body| be a new [=/map=] matching the <code>browsingContext.StartScreencastResult</code>
   with the <code>screencast</code> field set to |screencast| and
   <code>path</code> field set to |path|.

1. Return [=success=] with data |body|.

</div>

#### The browsingContext.stopScreencast Command ####  {#command-browsingContext-stopScreencast}

The <dfn export for=commands>browsingContext.stopScreencast</dfn> command
stops the screencast.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.StopScreencast = (
        method: "browsingContext.stopScreencast",
        params: browsingContext.StopScreencastParameters
      )

      browsingContext.StopScreencastParameters = {
        screencast: browsingContext.Screencast
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.StopScreencastResult = {
         path: text,
         ? error: text
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.stopScreencast">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |screencast| be the value of the "<code>screencast</code>" field in |command
   parameters|.

1. If |session|'s [=screencast recordings map=] does not <a for=map>contain</a>
   |screencast|, return [=error=] with [=error code=] [=no such screencast=].

1. Let |screencast recording| be |session|'s [=screencast recordings map=][|screencast|].

1. [=Stop a screencast recording=] given |screencast recording|.

1. [=map/Remove=] |screencast| from |session|'s [=screencast recordings map=].

1. Let |error| be |screencast recording|'s [=screencast recording/writeError=].

1. Let |path| be |screencast recording|'s [=screencast recording/path=].

1. Let |body| be a new [=/map=] matching the <code>browsingContext.StopScreencastResult</code>
   with the <code>path</code> field set to |path|, <code>error</code> field set to |error|
   if |error| is not null or omitted otherwise.

1. Return [=success=] with data |body|.

</div>

#### The browsingContext.traverseHistory Command ####  {#command-browsingContext-traverseHistory}

The <dfn export for=commands>browsingContext.traverseHistory</dfn> command
traverses the history of a given navigable by a delta.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      browsingContext.TraverseHistory = (
        method: "browsingContext.traverseHistory",
        params: browsingContext.TraverseHistoryParameters
      )

      browsingContext.TraverseHistoryParameters = {
        context: browsingContext.BrowsingContext,
        delta: js-int,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      browsingContext.TraverseHistoryResult = EmptyResult
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for browsingContext.traverseHistory">
The [=remote end steps=] with |command parameters| are:

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |command parameters|["<code>context</code>"].

1. If |navigable| is not a [=/top-level traversable=], return [=error=] with
   [=error code=] [=invalid argument=].

1. Assert: |navigable| is not null.

1. Let |delta| be |command parameters|["<code>delta</code>"].

1. Let |resume id| be a unique string.

1. [=Queue a task=] on |navigable|'s [=session history traversal queue=] to run
   the following steps:

    1. Let |all steps| be the result of [=getting all used history steps=] for |navigable|.

    1. Let |current index| be the index of |navigable|'s [=current session history
       step=] within |all steps|.

    1. Let |target index| be |current index| plus |delta|.

    1. Let |valid entry| be false if |all steps|[|target index|] does not exist,
       or true otherwise.

    1. [=Resume=] with "<code>check history</code>", |resume id|, and
       |valid entry|.

1. Let |is valid entry| be [=await=] with «"<code>check history</code>"», and
   |resume id|.

1. If |is valid entry| is false, return [=error=] with error code [=no such
   history entry=].

1. [=Traverse the history by a delta=] given |delta| and |navigable|.

   ISSUE: There is a race condition in the algorithm as written because by the
   time we try to navigate the target session history entry might not
   exist. Once we support waiting for history to navigate we can handle this
   more robustly.

1. TODO: Support waiting for the history traversal to complete.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.TraverseHistoryResult</code> production.

1. Return [=success=] with data |body|.

</div>

<div algorithm>

The <dfn export>WebDriver BiDi page show</dfn> steps given <var
ignore>context</var> and [=WebDriver BiDi navigation status|navigation status=] |navigation status|
are:

Issue: Do we want to expose a `browsingContext.pageShow event? In that case we'd
need to call this whenever `pageshow` is going to be emitted, not just on
bfcache restore, and also add the persisted status to the data.

 1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

 1. [=Resume=] with "<code>page show</code>", |navigation id|, and
    |navigation status|.

</div>

<div algorithm>

The <dfn export>WebDriver BiDi pop state</dfn> steps given <var
ignore>context</var> and [=WebDriver BiDi navigation status|navigation status=] |navigation status|
are:

 1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

 1. [=Resume=] with "<code>pop state</code>", |navigation id|, and
    |navigation status|.

</div>


### Events ### {#module-contexts-events}

#### The browsingContext.contextCreated Event #### {#event-browsingContext-contextCreated}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.ContextCreated = (
         method: "browsingContext.contextCreated",
         params: browsingContext.Info
        )
      </pre>
   </dd>
</dl>

<div algorithm>

To <dfn>Recursively emit context created events</dfn> given |session| and |navigable|:

1. [=Emit a context created event=] with |session| and |navigable|.

1. For each child navigable, |child|, of |navigable|:

  1. [=Recursively emit context created events=] given |session| and |child|.

</div>

<div algorithm>

To <dfn>Emit a context created event</dfn> given |session| and |navigable|:

1. Let |params| be the result of [=get the navigable info=] given
   |navigable|, 0, and true.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.ContextCreated</code> production, with the
   <code>params</code> field set to |params|.

1. [=Emit an event=] with |session| and |body|.

</div>


<div algorithm="remote end event trigger for browsingContext.contextCreated">

The [=remote end event trigger=] is
the <dfn export>WebDriver BiDi navigable created</dfn> steps given
[=/navigable=] |navigable| and [=/navigable=] |opener navigable|:

1. Set |navigable|'s [=original opener=] to |opener navigable|,
   if |opener navigable| is provided.

1. If the [=navigable cache behavior=] with |navigable| is "<code>bypass</code>",
   then perform implementation-defined steps to disable any implementation-specific
   resource caches for network requests originating from |navigable|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.contextCreated</code>" and |related navigables|:

  1. [=Emit a context created event=] given |session| and |navigable|.

</div>

<div algorithm="remote end subscribe steps for browsingContext.contextCreated">

The [=remote end subscribe steps=], with [=subscribe priority=] 1, given
|session|, |navigables| and <var ignore>include global</var> are:

1. For each |navigable| in |navigables|:

  1. [=Recursively emit context created events=] given |session| and |navigable|.

</div>

#### The browsingContext.contextDestroyed Event #### {#event-browsingContext-contextDestroyed}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.ContextDestroyed = (
         method: "browsingContext.contextDestroyed",
         params: browsingContext.Info
        )
      </pre>
   </dd>
</dl>
The [=remote end event trigger=] is:

<div algorithm="remote end event trigger for browsingContext.contextDestroyed">

The [=remote end event trigger=] is
the <dfn export>WebDriver BiDi navigable destroyed</dfn> steps given [=/navigable=] |navigable|:

1. Let |params| be the result of [=get the navigable info=], given
   |navigable|, null, and true.

1. Let |body| be a [=/map=] matching the
    <code>browsingContext.ContextDestroyed</code> production, with the
    <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|'s [=navigable/parent=],
   if that is not null, or an empty [=/set=] otherwise.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.contextDestroyed</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

  1. Let |subscriptions to remove| be a [=/set=].

  1. For each |subscription| in |session|'s [=subscriptions=]:

     1. If |subscription|'s [=subscription/top-level traversable ids=] [=set/contains=] |navigable|'s [=navigable id=];

        1. [=set/Remove=] |navigable|'s [=navigable id=] from |subscription|'s [=subscription/top-level traversable ids=].

        1. If |subscription|'s [=subscription/top-level traversable ids=] is empty:

           1. [=set/Append=] |subscription| to |subscriptions to remove|.

  1. [=list/Remove=] |subscriptions to remove| from |session|'s [=subscriptions=].

Issue: It's unclear if we ought to only fire this event for browsing
contexts that have active documents; navigation can also cause contexts to
become inaccessible but not yet get discarded because bfcache.
</div>

#### The browsingContext.navigationStarted Event #### {#event-browsingContext-navigationStarted}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.NavigationStarted = (
         method: "browsingContext.navigationStarted",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi navigation started</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

 1. Let |body| be a [=/map=] matching the
    <code>browsingContext.NavigationStarted</code> production, with the
    <code>params</code> field set to |params|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. [=Resume=] with "<code>navigation started</code>", |navigation id|, and
   |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.navigationStarted</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.fragmentNavigated Event #### {#event-browsingContext-fragmentNavigated}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.FragmentNavigated = (
         method: "browsingContext.fragmentNavigated",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi fragment navigated</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the
    <code>browsingContext.FragmentNavigated</code> production, with the
    <code>params</code> field set to |params|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |related navigable| be a [=/set=] containing |navigable|.

1. [=Resume=] with "<code>fragment navigated</code>", |navigation id|, and
   |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.fragmentNavigated</code>" and |related navigable|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.historyUpdated Event #### {#event-browsingContext-historyUpdated}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.HistoryUpdated = (
          method: "browsingContext.historyUpdated",
          params: browsingContext.HistoryUpdatedParameters
        )

        browsingContext.HistoryUpdatedParameters = {
          context: browsingContext.BrowsingContext,
          timestamp: js-uint,
          url: text,
          ? userContext: browser.UserContext
        }
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi history updated</dfn> steps
given [=/navigable=] |navigable|:

1. Let |url| be the result of running the [=URL serializer=],
   given |navigable|'s [=active browsing context=]'s [=active document=]'s <a spec=dom>URL</a>.

1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

1. Let |timestamp| be a [=time value=] representing the current date and time in UTC.

1. Let |params| be a [=/map=] matching the <code>browsingContext.HistoryUpdatedParameters</code> production,
   with the <code>url</code> field set to |url|,
   the <code>timestamp</code> field set to |timestamp|,
   the <code>context</code> field set to |navigable|'s [=navigable id=] and
   the <code>userContext</code> field set to |user context id|.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.HistoryUpdated</code> production, with the
   <code>params</code> field set to |params|.

1. Let |related browsing contexts| be a [=/set=] containing |navigable|'s [=active browsing context=].

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.historyUpdated</code>" and |related browsing contexts|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.domContentLoaded Event #### {#event-browsingContext-domContentLoaded}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.DomContentLoaded = (
         method: "browsingContext.domContentLoaded",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi DOM content loaded</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.DomContentLoaded</code> production, with the
   <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. [=Resume=] with "<code>domContentLoaded</code>", |navigation id|, and
   |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.domContentLoaded</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.load Event #### {#event-browsingContext-load}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.Load = (
         method: "browsingContext.load",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi load complete</dfn> steps given
[=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the <code>browsingContext.Load</code>
   production, with the <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. [=Resume=] with "<code>load</code>", |navigation id| and
   |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.load</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.downloadWillBegin Event #### {#event-browsingContext-downloadWillBegin}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.DownloadWillBegin = (
         method: "browsingContext.downloadWillBegin",
         params: browsingContext.DownloadWillBeginParams
        )

        browsingContext.DownloadWillBeginParams = {
          download: browsingContext.Download,
          suggestedFilename: text,
          browsingContext.BaseNavigationInfo
        }
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the
<dfn export>WebDriver BiDi download will begin</dfn> steps given [=/navigable=] |navigable| and
[=WebDriver BiDi navigation status|navigation status=] |navigation status|:

1. Let |navigation info| be the result of [=get the navigation info=] given
   |navigable| and |navigation status|.

1. Let |download| be the string representation of a [[!RFC9562|UUID]].

1. [=map/Set=] [=download id map=][|navigation status|'s [=WebDriver BiDi navigation status/downloadResponse=]] to |download|.

1. Let |params| be a [=/map=] matching the
   <code>browsingContext.DownloadWillBeginParams</code> production, with the
   <code>context</code> field set to |navigation info|["<code>context</code>"], the
   <code>navigation</code> field set to |navigation info|["<code>navigation</code>"],
   the <code>timestamp</code> field set to
   |navigation info|["<code>timestamp</code>"], the <code>url</code> field set to
   |navigation info|["<code>url</code>"], the <code>download</code> field set to |download|
   and the <code>suggestedFilename</code> field set to
   |navigation status|'s [=WebDriver BiDi navigation status/suggestedFilename=].

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.DownloadWillBegin</code> production, with the
   <code>params</code> field set to |params|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. [=Resume=] with "<code>download started</code>", |navigation id|, and
   |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.downloadWillBegin</code>" and |related navigables|:

   1. [=Emit an event=] with |session| and |body|.

1. Let |download behavior| be [=get download behavior=] with |navigable|.

1. Return |download behavior|.

</div>

#### The browsingContext.downloadEnd Event #### {#event-browsingContext-downloadEnd}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.DownloadEnd = (
          method: "browsingContext.downloadEnd",
          params: browsingContext.DownloadEndParams
        )

        browsingContext.DownloadEndParams = {
          (
            browsingContext.DownloadCanceledParams //
            browsingContext.DownloadCompleteParams
          )
        }

        browsingContext.DownloadCanceledParams = (
          status: "canceled",
          download: browsingContext.Download,
          browsingContext.BaseNavigationInfo
        )

        browsingContext.DownloadCompleteParams = (
          status: "complete",
          download: browsingContext.Download,
          filepath: text / null,
          browsingContext.BaseNavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi download end</dfn> steps given
[=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |navigation info| be the result of [=get the navigation info=] given
   |navigable| and |navigation status|.

1. Assert |navigation info|["<code>status</code>"] is equal to either
   "<code>complete</code>" or "<code>canceled</code>".

1. Let |download response| be |navigation status|'s [=WebDriver BiDi navigation status/downloadResponse=].

1. If [=download id map=] [=map/contains=] |download response|, let |download| be
   [=download id map=][|download response|], otherwise let |download| be
   the string representation of a [[!RFC9562|UUID]].

1. If |navigation info|["<code>status</code>"] is "<code>complete</code>", let
   |params| be a [=/map=] matching the
   <code>browsingContext.DownloadCompleteParams</code> production, with the <code>download</code> field
   set to |download|, the <code>filepath</code> field set to
   |navigation status|'s [=WebDriver BiDi navigation status/downloadedFilepath=], the
   <code>context</code> field set to |navigation info|["<code>context</code>"], the
   <code>navigation</code> field set to |navigation info|["<code>navigation</code>"],
   the <code>timestamp</code> field set to
   |navigation info|["<code>timestamp</code>"], and the <code>url</code> field set to
   |navigation info|["<code>url</code>"].

   Note: <code>filepath</code> can be null for completed downloads if the filepath is
   not available for whatever reason.

1. Otherwise, let |params| be a [=/map=] matching the
   <code>browsingContext.DownloadCanceledParams</code> production, with the <code>download</code> field
   set to |download|, the <code>context</code> field set to |navigation info|["<code>context</code>"],
   the <code>navigation</code> field set to |navigation info|["<code>navigation</code>"],
   the <code>timestamp</code> field set to
   |navigation info|["<code>timestamp</code>"], and the <code>url</code> field set to
   |navigation info|["<code>url</code>"].

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.DownloadEnd</code> production, with the
   <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.downloadEnd</code>" and |related navigables|:

   1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.navigationAborted Event #### {#event-browsingContext-navigationAborted}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.NavigationAborted = (
         method: "browsingContext.navigationAborted",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi navigation aborted</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the
    <code>browsingContext.NavigationAborted</code> production, with the
    <code>params</code> field set to |params|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. [=Resume=] with "<code>navigation aborted</code>", |navigation id|, and |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.navigationAborted</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.navigationCommitted Event #### {#event-browsingContext-navigationCommitted}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.NavigationCommitted = (
         method: "browsingContext.navigationCommitted",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi navigation committed</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the
    <code>browsingContext.NavigationCommitted</code> production, with the
    <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. [=Resume=] with "<code>navigation committed</code>", |navigation id|, and |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.navigationCommitted</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>


#### The browsingContext.navigationFailed Event #### {#event-browsingContext-navigationFailed}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.NavigationFailed = (
         method: "browsingContext.navigationFailed",
         params: browsingContext.NavigationInfo
        )
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi navigation failed</dfn> steps
given [=/navigable=] |navigable| and [=WebDriver BiDi navigation status|navigation status=]
|navigation status|:

1. Let |params| be the result of [=get the navigation info=] given |navigable|
   and |navigation status|.

1. Let |body| be a [=/map=] matching the
    <code>browsingContext.NavigationFailed</code> production, with the
    <code>params</code> field set to |params|.

1. Let |navigation id| be |navigation status|'s [=WebDriver BiDi navigation status/id=].

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. [=Resume=] with "<code>navigation failed</code>", |navigation id|, and |navigation status|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.navigationFailed</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.userPromptClosed Event #### {#event-browsingContext-userPromptClosed}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.UserPromptClosed = (
          method: "browsingContext.userPromptClosed",
          params: browsingContext.UserPromptClosedParameters
        )

        browsingContext.UserPromptClosedParameters = {
          context: browsingContext.BrowsingContext,
          accepted: bool,
          type: browsingContext.UserPromptType,
          ? userContext: browser.UserContext,
          ? userText: text
        }
      </pre>
   </dd>
</dl>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi user prompt closed</dfn> steps
given {{Window}} |window|, string |type|, boolean |accepted| and optional text |user text|
(default: null).

1. Let |navigable| be |window|'s [=window/navigable=].

1. Let |navigable id| be the [=navigable id=] for |navigable|.

1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

1. Let |params| be a [=/map=] matching the
   <code>browsingContext.UserPromptClosedParameters</code> production with the
   <code>context</code> field set to |navigable id|, the <code>userContext</code>
   field set to |user context id|, the <code>accepted</code> field set to |accepted|,
   the <code>type</code> field set to |type|, and the <code>userText</code> field
   set to |user text| if |user text| is not null or omitted otherwise.

1. Let |body| be a [=/map=] matching the
   <code>BrowsingContextUserPromptClosedEvent</code> production, with the
   <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.userPromptClosed</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The browsingContext.userPromptOpened Event #### {#event-browsingContext-userPromptOpened}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        browsingContext.UserPromptOpened = (
          method: "browsingContext.userPromptOpened",
          params: browsingContext.UserPromptOpenedParameters
        )

        browsingContext.UserPromptOpenedParameters = {
          context: browsingContext.BrowsingContext,
          handler: session.UserPromptHandlerType,
          message: text,
          type: browsingContext.UserPromptType,
          ? userContext: browser.UserContext,
          ? defaultValue: text
        }
      </pre>
   </dd>
</dl>

<div algorithm>
To <dfn>get navigable's user prompt handler</dfn> given |type| and |navigable|:

1. Let |user context| be |navigable|'s [=associated user context=].

1. If [=unhandled prompt behavior overrides map=] [=map/contains=] |user context|:

  1. Let |unhandled prompt behavior override| be [=unhandled prompt behavior overrides map=][|user context|].

  1. If |unhandled prompt behavior override|[|type|] is not null, return
     |unhandled prompt behavior override|[|type|].

  1. If |unhandled prompt behavior override|[<code>"default"</code>] is not null,
     return |unhandled prompt behavior override|[<code>"default"</code>].

1. Let |handler configuration| be [=get the prompt handler=] with |type|.

1. Return |handler configuration|'s [=prompt handler configuration/handler=].

</div>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi user prompt opened</dfn> steps
given {{Window}} |window|, string |type|, string |message|, and optional text |default value|
(default: null).

1. Let |navigable| be |window|'s [=window/navigable=].

1. Let |navigable id| be the [=navigable id=] for |navigable|.

1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

1. Let |handler| be [=get navigable's user prompt handler=] with |type| and
   |navigable|.

1. Let |params| be a [=/map=] matching the
   <code>browsingContext.UserPromptOpenedParameters</code> production with the
   <code>context</code> field set to |navigable id|, the <code>userContext</code>
   field set to |user context id|, the <code>type</code> field set to |type|,
   the <code>message</code> field set to |message|, the <code>defaultValue</code>
   field set to |default value| if |default value| is not null or omitted otherwise,
   and the <code>handler</code> field set to |handler|.

1. Let |body| be a [=/map=] matching the
   <code>browsingContext.UserPromptOpened</code> production, with the
   <code>params</code> field set to |params|.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>browsingContext.userPromptOpened</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

1. If |handler| is "<code>ignore</code>", set handler to "<code>none</code>".

1. Return |handler|.

</div>

## The emulation Module ## {#module-emulation}

The <dfn export for=modules>emulation</dfn> module contains commands and events
relating to emulation of browser APIs.

### Definition ### {#module-emulation-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
EmulationCommand = (
  emulation.SetForcedColorsModeThemeOverride //
  emulation.SetGeolocationOverride //
  emulation.SetLocaleOverride //
  emulation.SetMediaFeaturesOverride //
  emulation.SetNetworkConditions //
  emulation.SetScreenOrientationOverride //
  emulation.SetScreenSettingsOverride //
  emulation.SetScriptingEnabled //
  emulation.SetScrollbarTypeOverride //
  emulation.SetTimezoneOverride //
  emulation.SetTouchOverride //
  emulation.SetUserAgentOverride //
  emulation.SetViewportMetaOverride
)

</pre>

<pre class="cddl" data-cddl-module="local-cddl">
EmulationResult = (
  emulation.SetForcedColorsModeThemeOverrideResult /
  emulation.SetGeolocationOverrideResult /
  emulation.SetLocaleOverrideResult /
  emulation.SetMediaFeaturesOverrideResult /
  emulation.SetScreenOrientationOverrideResult /
  emulation.SetScriptingEnabledResult /
  emulation.SetScrollbarTypeOverrideResult /
  emulation.SetTimezoneOverrideResult /
  emulation.SetTouchOverrideResult /
  emulation.SetUserAgentOverrideResult /
  emulation.SetViewportMetaOverrideResult
)
</pre>

A [=BiDi session=] has an <dfn for=session>emulated user agent</dfn> which is a
[=struct=] with an [=struct/item=] named
<dfn for="emulated user agent">default user agent</dfn>, which is a string or null,
an [=struct/item=] named
<dfn for="emulated user agent">user context user agent</dfn>, which is a weak map
between [=user context|user contexts=] and string, and an [=struct/item=] named
<dfn for="emulated user agent">navigable user agent</dfn>, which is a weak map
between [=/navigables=] and string.

A [=BiDi session=] has <dfn for=session>emulated maxTouchPoints</dfn>, which is a [=struct=] with
an [=struct/item=] named <dfn for="emulated maxTouchPoints">default</dfn>, which is an integer or null, initially null;
an [=struct/item=] named <dfn for="emulated maxTouchPoints">user contexts</dfn>, which is a weak map between [=user context|user contexts=] and integer, initially empty;
and an [=struct/item=] named <dfn for="emulated maxTouchPoints">navigables</dfn>, which is a weak map between [=/navigables=] and integer, initially empty.

A <dfn>screen orientation override</dfn> is a [=struct=] with:

* [=struct/item=] named <dfn attribute for="screen orientation override">natural</dfn> which is a string;
* [=struct/item=] named <dfn attribute for="screen orientation override">type</dfn> which is a string;

A [=remote end=] has a <dfn>screen orientation overrides map</dfn> which is a weak
map between [=user context|user contexts=] and [=screen orientation override=].

### Commands ### {#module-emulation-commands}
#### The emulation.setForcedColorsModeThemeOverride Command ####  {#command-emulation-setForcedColorsModeThemeOverride}

The <dfn export for=commands>emulation.setForcedColorsModeThemeOverride</dfn> command modifies
[=forced colors mode=] theming characteristics on the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetForcedColorsModeThemeOverride = (
        method: "emulation.setForcedColorsModeThemeOverride",
        params: emulation.SetForcedColorsModeThemeOverrideParameters
      )

      emulation.SetForcedColorsModeThemeOverrideParameters = {
        theme: emulation.ForcedColorsModeTheme / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }

      emulation.ForcedColorsModeTheme = "light" / "dark"
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetForcedColorsModeThemeOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

Note: Check out the {{ForcedColorsModeAutomationTheme}} for the corresponding enum mapping in the CSS specification.

A [=remote end=] has a <dfn>forced colors mode theme override configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] string.

<div algorithm>
To <dfn>update emulated forced colors theme</dfn> for [=/navigable=] |navigable|:

1. Let |theme| be the result of [=get WebDriver configuration value=] of
   [=forced colors mode theme override configuration=] for |navigable|.

1. Assert: |theme| is "<code>light</code>", "<code>dark</code>" or
   [=WebDriver configuration/unset=].

1. If |theme| is [=WebDriver configuration/unset=], set |theme| to "<code>none</code>".

1. [=Set emulated forced colors theme data=] with |navigable| and |theme|.

</div>

<div algorithm="remote end steps for emulation.setForcedColorsModeThemeOverride">

The [=remote end steps=] with |command parameters| are:

1. Let |theme| be |command parameters|["<code>theme</code>"].

1. If |theme| is null, set |theme| to [=WebDriver configuration/unset=].

1. If the implementation does not support setting |theme|, then return [=error=] with [=error code=]
   [=unsupported operation=].

1. Let |affected navigables| be the result of [=trying=] to [=store WebDriver configuration=]
   [=forced colors mode theme override configuration=] |theme| for |command parameters|.

1. For each |navigable| of |affected navigables|:

   1. [=Update emulated forced colors theme=] for |navigable|.

1. Return [=success=] with data null.

</div>

#### The emulation.setGeolocationOverride Command ####  {#command-emulation-setGeolocationOverride}

The <dfn export for=commands>emulation.setGeolocationOverride</dfn> command modifies geolocation characteristics on the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetGeolocationOverride = (
        method: "emulation.setGeolocationOverride",
        params: emulation.SetGeolocationOverrideParameters
      )

      emulation.SetGeolocationOverrideParameters = {
        (
          (coordinates: emulation.GeolocationCoordinates / null) //
          (error: emulation.GeolocationPositionError)
        ),
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }

      emulation.GeolocationCoordinates = {
         latitude: -90.0..90.0,
         longitude: -180.0..180.0,
         ? accuracy: (float .ge 0.0) .default 1.0,
         ? altitude: float / null .default null,
         ? altitudeAccuracy: (float .ge 0.0) / null .default null,
         ? heading: (0.0...360.0) / null .default null,
         ? speed: (float .ge 0.0) / null .default null,
      }

      emulation.GeolocationPositionError = {
         type: "positionUnavailable"
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetGeolocationOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

A <dfn>geolocation override</dfn> is a [=struct=] with:
* [=struct/item=] named <dfn attribute for="geolocation override">latitude</dfn> which is a float;
* [=struct/item=] named <dfn attribute for="geolocation override">longitude</dfn> which is a float;
* [=struct/item=] named <dfn attribute for="geolocation override">accuracy</dfn> which is a float;
* [=struct/item=] named <dfn attribute for="geolocation override">altitude</dfn> which is a float or null;
* [=struct/item=] named <dfn attribute for="geolocation override">altitudeAccuracy</dfn> which is a float or null;
* [=struct/item=] named <dfn attribute for="geolocation override">heading</dfn> which is a float or null;
* [=struct/item=] named <dfn attribute for="geolocation override">speed</dfn> which is a float or null.

A [=remote end=] has a <dfn>geolocation override configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] [=geolocation override=].

<div algorithm>
To <dfn>update geolocation override</dfn> for [=/navigable=] |navigable|:

1. Let |emulated position data| be the result of [=get WebDriver configuration value=] of
   [=geolocation override configuration=] for |navigable|.

1. If |emulated position data| is [=WebDriver configuration/unset=], set |emulated position data| to
   null.

1. [=Set emulated position data=] with |navigable| and |emulated position data|.

</div>

<div algorithm="remote end steps for emulation.setGeolocationOverride">

The [=remote end steps=] with |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>coordinates</code>" and
   |command parameters|["<code>coordinates</code>"] [=map/contains=]
   "<code>altitudeAccuracy</code>" and
   |command parameters|["<code>coordinates</code>"] doesn't [=map/contain=]
   "<code>altitude</code>", return [=error=] with [=error code=]
   [=invalid argument=].

1. If |command parameters| [=map/contains=] "<code>error</code>":

   1. Assert |command parameters|["<code>error</code>"]["<code>type</code>"] equals
      "<code>positionUnavailable</code>".

   1. Let |emulated position data| be a [=/map=] matching [=GeolocationPositionError=]
      production, with <code>code</code> field set to [=POSITION_UNAVAILABLE=] and
      <code>message</code> field set to the empty string.

      Note: <code>message</code> will be ignored by implementation according to the geolocation spec.

1. Otherwise, let |emulated position data| be |command parameters|["<code>coordinates</code>"].

1. If |emulated position data| is null, set |emulated position data| to
   [=WebDriver configuration/unset=].

1. Let |affected navigables| be the result of [=trying=] to [=store WebDriver configuration=]
   [=geolocation override configuration=] |emulated position data| for |command parameters|.

1. For each |navigable| of |affected navigables|:

   1. [=Update geolocation override=] for |navigable|.

1. Return [=success=] with data null.

</div>

#### The emulation.setLocaleOverride Command ####  {#command-emulation-setLocaleOverride}

The <dfn export for=commands>emulation.setLocaleOverride</dfn> command modifies
locale on the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetLocaleOverride = (
        method: "emulation.setLocaleOverride",
        params: emulation.SetLocaleOverrideParameters
      )

      emulation.SetLocaleOverrideParameters = {
        locale: text / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetLocaleOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>

The <dfn export>WebDriver BiDi emulated language</dfn> steps given an [=environment settings object=]
|environment settings|:

1. Let |related navigables| be the result of [=get related navigables=] given
   |environment settings|.

1. For each |navigable| of |related navigables|:

    1. Let |top-level traversable| be |navigable|’s
       [=navigable/top-level traversable=].

    1. Let |user context| be |top-level traversable|'s [=associated user context=].

    1. If [=locale overrides map=] [=map/contains=] |top-level traversable|, return
       [=locale overrides map=][|top-level traversable|].

    1. If [=locale overrides map=] [=map/contains=] |user context|, return
       [=locale overrides map=][|user context|].

1. Return null

</div>

<div algorithm="remote end steps for emulation.setLocaleOverride">

The [=remote end steps=] with |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated locale| be |command parameters|["<code>locale</code>"].

1. If |emulated locale| is not null and [=IsStructurallyValidLanguageTag=](|emulated locale|)
   returns false, return [=error=] with [=error code=] [=invalid argument=].

1. Let |navigables| be a [=/set=].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

1. Otherwise:

   1. Assert the <code>userContexts</code> field of |command parameters| is present.

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated locale| is null, [=map/remove=] |user context| from [=locale overrides map=].

      1. Otherwise, [=map/set=] [=locale overrides map=][|user context|] to |emulated locale|.

      1. [=list/For each=] |top-level traversable| of the list of all
         [=/top-level traversables=] whose [=associated user context=] is
         |user context|:

         1. [=list/Append=] |top-level traversable| to |navigables|.

1. For each |navigable| of |navigables|:

   1. If |emulated locale| is null, [=map/remove=] |navigable| from [=locale overrides map=].

   1. Otherwise, [=map/set=] [=locale overrides map=][|navigable|] to |emulated locale|.

1. Return [=success=] with data null.

</div>

#### The emulation.setMediaFeaturesOverride Command ####  {#command-emulation-setMediaFeaturesOverride}

The <dfn export for=commands>emulation.setMediaFeaturesOverride</dfn> command
allows overriding the values of various media features.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetMediaFeaturesOverride = (
        method: "emulation.setMediaFeaturesOverride",
        params: emulation.SetMediaFeaturesOverrideParameters
      )

      emulation.SetMediaFeaturesOverrideParameters = {
        features: emulation.MediaFeatures / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }

      emulation.MediaFeatures = [emulation.MediaFeature]

      emulation.MediaFeature = {
         name: text
         value: text
      }

    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetMediaFeaturesOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

A [=remote end=] has a <dfn>media features override configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] [=/map=].

<div algorithm>
To get <dfn export>WebDriver BiDi media feature value</dfn> for [=/Document=] |document| and media
feature name |name|:

1. Let |navigable| be |document|'s [=/node navigable=].

1. Let |media features override map| be the result of [=get WebDriver configuration value=] of
   [=media features override configuration=] for |navigable|.

1. If |media features override map| is not [=WebDriver configuration/unset=] and
   |media features override map| [=map/contains=] |name|, return
   |media features override map|[|name|].

1. Return null.

</div>

<div algorithm="remote end steps for emulation.setMediaFeaturesOverride">

The [=remote end steps=] with |command parameters| are:

1. Let |media features override| be |command parameters|["<code>features</code>"].

1. If |media features override| is null, set |media features override| to
   [=WebDriver configuration/unset=].

1. Let |media features override map| be an empty [=/map=].

1. For each |media feature| of |media features override|:
   1. [=map/Set=] |media features override map|[|media feature|["<code>name</code>"]] to
      |media feature|["<code>value</code>"].

1. If the implementation does not support overriding any of the media features in
   |media features override map|, return [=error=] with [=error code=] [=unsupported operation=].

1. Let |affected navigables| be the result of [=store WebDriver configuration=]
   [=media features override configuration=] |media features override map| for |command parameters|.

1. For each |navigable| of |affected navigables|:
   1. [=list/For each=] |document| currently loaded in |navigable| or its [=descendant navigables=]:
      1. Run [=evaluate media queries and report changes=] for |document|.

1. Return [=success=] with data null.

</div>

#### The emulation.setNetworkConditions Command #### {#command-emulation-setNetworkConditions}

The <dfn export for=commands>emulation.setNetworkConditions</dfn> command
emulates specific network conditions for the given browsing context or for a user
context.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetNetworkConditions = (
        method: "emulation.setNetworkConditions",
        params: emulation.SetNetworkConditionsParameters
      )

      emulation.SetNetworkConditionsParameters = {
        networkConditions: emulation.NetworkConditions / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }

      emulation.NetworkConditions = emulation.NetworkConditionsOffline

      emulation.NetworkConditionsOffline = {
        type: "offline"
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetNetworkConditionsResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>

To <dfn>apply network conditions</dfn>:

1. For each [=WebSocket=] object |webSocket|:

  1. Let |realm| be |webSocket|'s [=relevant Realm=].

  1. Let |environment settings| be the [=environment settings object=] whose
     [=realm execution context=]'s Realm component is |realm|.

  1. If the result of [=WebDriver BiDi network is offline=] with |environment settings| is true:

     1. [=Fail the WebSocket connection=] |webSocket|.

1. For each [=WebTransport=] object |webTransport|:

  1. Let |realm| be |webSocket|'s [=relevant Realm=].

  1. Let |environment settings| be the [=environment settings object=] whose
     [=realm execution context=]'s Realm component is |realm|.

  1. If the result of [=WebDriver BiDi network is offline=] with |environment settings| is true:

     1. [=Cleanup WebTransport=] |webTransport|.

</div>

<div algorithm="remote end steps for emulation.setNetworkConditions">

The [=remote end steps=] with |command parameters| and |session| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>context</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated network conditions| be null.

1. If |command parameters|["<code>networkConditions</code>"] is not null and
      |command parameters|["<code>networkConditions</code>"]["<code>type</code>"]
      equals "<code>offline</code>", set |emulated network conditions| to a new
      [=emulated network conditions struct=] with
      [=emulated-network-conditions-struct/offline=] set to true.

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

   1. For each |navigable| of |navigables|:

      1. If |emulated network conditions| is null, [=map/remove=] |navigable| from
         |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/navigable network conditions=]

      1. Otherwise, [=map/set=] |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/navigable network conditions=][|navigable|] to
         |emulated network conditions|.

1. If the <code>userContexts</code> field of |command parameters| is present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated network conditions| is null, [=map/remove=] |user context| from
         |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/user context network conditions=].

      1. Otherwise, [=map/set=] |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/user context network conditions=][|user context|]
         to |emulated network conditions|.

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>context</code>",
   [=map/set=] |session|'s [=emulated network conditions=]'s
   [=emulated network conditions/default network conditions=] to |emulated network conditions|.

1. [=Apply network conditions=].

1. Return [=success=] with data null.

</div>

#### The emulation.setScreenSettingsOverride Command ####  {#command-emulation-setScreenSettingsOverride}

The <dfn export for=commands>emulation.setScreenSettingsOverride</dfn> command
emulates [=web-exposed screen area=] and [=web-exposed available screen area=] of the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetScreenSettingsOverride = (
        method: "emulation.setScreenSettingsOverride",
        params: emulation.SetScreenSettingsOverrideParameters
      )

      emulation.ScreenArea = {
        width: js-uint,
        height: js-uint
      }

      emulation.SetScreenSettingsOverrideParameters = {
        screenArea: emulation.ScreenArea / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetScreenSettingsOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>

The <dfn export>WebDriver BiDi emulated available screen area</dfn> steps given [=/navigable=] |navigable|:

1. Let |top-level traversable| be |navigable|’s [=navigable/top-level traversable=].

1. Let |user context| be |top-level traversable|'s [=associated user context=].

1. If [=screen settings overrides=] [=map/contains=] |top-level traversable|, return
   [=screen settings overrides=][|top-level traversable|].

1. If [=screen settings overrides=] [=map/contains=] |user context|, return
   [=screen settings overrides=][|user context|].

1. Return null

</div>

<div algorithm>

The <dfn export>WebDriver BiDi emulated total screen area</dfn> steps given [=/navigable=] |navigable|:

1. Let |top-level traversable| be |navigable|’s [=navigable/top-level traversable=].

1. Let |user context| be |top-level traversable|'s [=associated user context=].

1. If [=screen settings overrides=] [=map/contains=] |top-level traversable|, return
   [=screen settings overrides=][|top-level traversable|].

1. If [=screen settings overrides=] [=map/contains=] |user context|, return
   [=screen settings overrides=][|user context|].

1. Return null

</div>

<div algorithm="remote end steps for emulation.setScreenSettingsOverride">

The [=remote end steps=] with |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated screen area| be |command parameters|["<code>screenArea</code>"].

1. If |emulated screen area| is not null:

   1. [=map/Set=] |emulated screen area|["<code>x</code>"] to 0.

   1. [=map/Set=] |emulated screen area|["<code>y</code>"] to 0.

1. Let |navigables| be a [=/set=].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

   1. Let |target| be [=screen settings overrides/navigable screen settings=].

   1. For each |navigable| of |navigables|:

      1. If |emulated screen area| is null, [=map/remove=] |navigable|
         from |target|.

      1. Otherwise, [=map/set=] |target|[|navigable|]
         to |emulated screen area|.

   1. Return [=success=] with data null.

1. Otherwise:

   1. Assert the <code>userContexts</code> field of |command parameters| is present.

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. Let |target| be [=screen settings overrides/user context screen settings=].

   1. For each |user context| of |user contexts|:

      1. If |emulated screen area| is null, [=map/remove=] |user context| from
         |target|.

      1. Otherwise, [=map/set=] |target|[|user context|] to |emulated screen area|.

   1. Return [=success=] with data null.

</div>

#### The emulation.setScreenOrientationOverride Command ####  {#command-emulation-setScreenOrientationOverride}

The <dfn export for=commands>emulation.setScreenOrientationOverride</dfn> command
emulates [=screen orientation=] of the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetScreenOrientationOverride = (
        method: "emulation.setScreenOrientationOverride",
        params: emulation.SetScreenOrientationOverrideParameters
      )

      emulation.ScreenOrientationNatural = "portrait" / "landscape"
      emulation.ScreenOrientationType = "portrait-primary" / "portrait-secondary" / "landscape-primary" / "landscape-secondary"

      emulation.ScreenOrientation = {
        natural: emulation.ScreenOrientationNatural,
        type: emulation.ScreenOrientationType
      }

      emulation.SetScreenOrientationOverrideParameters = {
        screenOrientation: emulation.ScreenOrientation / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetScreenOrientationOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>

To <dfn>set emulated screen orientation</dfn> given |navigable| and
|emulated screen orientation|:

Issue: Move this algorithm to screen orientation specification.

1. If |emulated screen orientation| is null:

   1. Set |navigable|'s [=current orientation angle=] to
      implementation-defined default.

   1. Set |navigable|'s [=current orientation type=] to
      implementation-defined default.

1. Otherwise:

   1. Let |emulated orientation type| be
      |emulated screen orientation|["<code>type</code>"].

   1. Let |emulated orientation angle| be the angle associated with
      |emulated orientation type| for screens with
      |emulated screen orientation|["<code>natural</code>"] orientations as defined
      in [=screen orientation values lists=].

   1. Set [=current orientation angle=] to |emulated orientation angle|.

   1. Set [=current orientation type=] to |emulated orientation type|.

1. Run the [=screen orientation change steps=] with the |navigable|'s
   [=active document=].

</div>

<div algorithm="remote end steps for emulation.setScreenOrientationOverride">

The [=remote end steps=] with |command parameters| are:

1. If the implementation is unable to adjust the [=screen orientations=]
   parameters with the given |command parameters| for any reason, return
   [=error=] with [=error code=] [=unsupported operation=].

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated screen orientation| be
   |command parameters|["<code>screenOrientation</code>"].

1. Let |navigables| be a [=/set=].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

1. Otherwise, if the <code>userContexts</code> field of |command parameters| is
   present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated screen orientation| is null, [=map/remove=] |user context| from
         [=screen orientation overrides map=].

      1. Otherwise, [=map/set=] [=screen orientation overrides map=][|user context|] to
         |emulated screen orientation|.

      1. [=list/For each=] |top-level traversable| of the list of all
         [=/top-level traversables=] whose [=associated user context=] is
         |user context|:

         1. [=list/Append=] |top-level traversable| to |navigables|.

1. For each |navigable| of |navigables|:

   1. Let |user context| be |navigable|'s [=associated user context=].

   1. If |emulated screen orientation| is null and [=screen orientation overrides map=]
      [=map/contains=] |user context|, [=set emulated screen orientation=] with |navigable| and
      [=screen orientation overrides map=][|user context|].

   1. Otherwise, [=set emulated screen orientation=] with |navigable| and
      |emulated screen orientation|.

1. Return [=success=] with data null.

</div>

#### The emulation.setUserAgentOverride Command ####  {#command-emulation-setUserAgentOverride}

The <dfn export for=commands>emulation.setUserAgentOverride</dfn> command modifies
User-Agent on the given top-level traversables, user contexts, or globally.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetUserAgentOverride = (
        method: "emulation.setUserAgentOverride",
        params: emulation.SetUserAgentOverrideParameters
      )

      emulation.SetUserAgentOverrideParameters = {
        userAgent: text / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetUserAgentOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
The <dfn export>WebDriver BiDi emulated User-Agent</dfn> steps given
[=environment settings object=] |environment settings| are:

1. Let |related navigables| be the result of [=get related navigables=] with
   |environment settings|.

1. For each |navigable| or |related navigables|:

   1. Let |top-level navigable| be |navigable|'s [=navigable/top-level traversable=].

   1. Let |user context| be |top-level navigable|'s [=associated user context=].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=emulated user agent=]'s [=navigable user agent=] contains
         |top-level navigable|, return |session|'s [=emulated user agent=]'s
         [=navigable user agent=][|top-level navigable|].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=emulated user agent=]'s [=user context user agent=]
         contains |user context|, return |session|'s [=emulated user agent=]'s
         [=user context user agent=][|user context|].

1. For each |session| in [=active BiDi sessions=]:

   1. Let |default emulated user agent| be |session|'s [=emulated user agent=]'s
      [=default user agent=].

   1. If |default emulated user agent| is not null, return
      |default emulated user agent|.

1. Return null.

</div>

<div algorithm="remote end steps for emulation.setUserAgentOverride">

The [=remote end steps=] given |session| and |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated user agent| be |command parameters|["<code>userAgent</code>"].

1. If |command parameters| [=map/contains=] "<code>contexts</code>":

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

   1. For each |navigable| of |navigables|:

      1. If |emulated user agent| is null, [=map/remove=] |navigable| from
         |session|'s [=emulated user agent=]'s [=navigable user agent=].

      1. Otherwise, [=map/set=] |session|'s [=emulated user agent=]'s
         [=navigable user agent=][|navigable|] to |emulated user agent|.

   1. Return [=success=] with data null.

1. If |command parameters| [=map/contains=] "<code>userContexts</code>":

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated user agent| is null, [=map/remove=] |user context| from
         |session|'s [=emulated user agent=]'s [=user context user agent=].

      1. Otherwise, [=map/set=] |session|'s [=emulated user agent=]'s
         [=user context user agent=][|user context|] to |emulated user agent|.

   1. Return [=success=] with data null.

1. [=map/Set=] |session|'s [=emulated user agent=]'s [=default user agent=] to
   |emulated user agent|.

1. Return [=success=] with data null.

</div>

#### The emulation.setViewportMetaOverride Command #### {#command-emulation-setViewportMetaOverride}

The <dfn export for=commands>emulation.setViewportMetaOverride</dfn> command modifies whether the browser respects
the <code>&lt;meta name=viewport&gt;</code> tag.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetViewportMetaOverride = (
        method: "emulation.setViewportMetaOverride",
        params: emulation.SetViewportMetaOverrideParameters
      )

      emulation.SetViewportMetaOverrideParameters = {
        viewportMeta: true / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetViewportMetaOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

A [=remote end=] has a <dfn>viewport meta override configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] boolean.

<div algorithm>

The <dfn export>WebDriver BiDi viewport meta state</dfn> steps given
[=/Document=] |document| are:

1. Let |navigable| be |document|'s [=node navigable=].

1. Let |viewport meta override| be the result of [=get WebDriver configuration value=] of
   [=viewport meta override configuration=] for |navigable|.

1. If |viewport meta override| is [=WebDriver configuration/unset=], return null.

1. Return |viewport meta override|.

</div>

<div algorithm="remote end steps for emulation.setViewportMetaOverride">

The [=remote end steps=] given |command parameters| are:

1. Let |viewport meta override| be |command parameters|["<code>viewportMeta</code>"].

1. If |viewport meta override| is null, set |viewport meta override| to
   [=WebDriver configuration/unset=].

1. Let |affected navigables| be the result of [=trying=] to [=store WebDriver configuration=]
   [=viewport meta override configuration=] |viewport meta override| for |command parameters|.

1. For each |navigable| of |affected navigables|, run [=evaluate media queries and report changes=]
   for [=/document=] currently loaded in a specified |navigable|.

1. Return [=success=] with data null.

</div>


#### The emulation.setScriptingEnabled Command ####  {#command-emulation-setScriptingEnabled}

The <dfn export for=commands>emulation.setScriptingEnabled</dfn> command emulates
disabling JavaScript on web pages.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetScriptingEnabled = (
        method: "emulation.setScriptingEnabled",
        params: emulation.SetScriptingEnabledParameters
      )

      emulation.SetScriptingEnabledParameters = {
        enabled: false / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetScriptingEnabledResult = EmptyResult
      </pre>
   </dd>
</dl>

Note: only emulation of disabled Javascript is supported.

<div algorithm>
The <dfn export>WebDriver BiDi scripting is enabled</dfn> steps given
[=environment settings object=] |settings| are:

1. Let |navigable| be |settings|'s [=relevant global object=]'s
   <a>associated <code>Document</code></a>'s [=/node navigable=].

1. Let |top-level traversable| be |navigable|’s [=navigable/top-level traversable=].

1. If [=scripting enabled overrides map=] contains |top-level traversable|, return
   [=scripting enabled overrides map=][|top-level traversable|]

1. Let |user context| be  |top-level traversable|'s [=associated user context=].

1. If [=scripting enabled overrides map=] contains |user context|, return
   [=scripting enabled overrides map=][|user context|].

1. Return true.

</div>

<div algorithm="remote end steps for emulation.setScriptingEnabled">

The [=remote end steps=] with |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated scripting enabled status| be |command parameters|["<code>enabled</code>"].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

   1. For each |navigable| of |navigables|:

      1. If |emulated scripting enabled status| is null, [=map/remove=] |navigable| from
         [=scripting enabled overrides map=].

      1. Otherwise, [=map/set=] [=scripting enabled overrides map=][|navigable|] to
         |emulated scripting enabled status|.

1. If the <code>userContexts</code> field of |command parameters| is present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated scripting enabled status| is null, [=map/remove=] |user context| from
         [=scripting enabled overrides map=].

      1. Otherwise [=map/set=] [=scripting enabled overrides map=][|user context|] to
         |emulated scripting enabled status|.

1. Return [=success=] with data null.

</div>

#### The emulation.setScrollbarTypeOverride Command #### {#command-emulation-setScrollbarTypeOverride}

The <dfn export for=commands>emulation.setScrollbarTypeOverride</dfn> command modifies
scrollbar type on the given top-level traversables, user contexts or globally.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetScrollbarTypeOverride = (
        method: "emulation.setScrollbarTypeOverride",
        params: emulation.SetScrollbarTypeOverrideParameters
      )

      emulation.SetScrollbarTypeOverrideParameters = {
        scrollbarType: "classic" / "overlay" / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetScrollbarTypeOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

A [=remote end=] has a <dfn>scrollbar type override configuration</dfn>, which is
[=WebDriver configuration=] with [=WebDriver configuration/associated type=] string.

<div algorithm>
To <dfn>update scrollbar type override</dfn> for [=/navigable=] |navigable|:

1. Let |scrollbar type override| be the result of [=get WebDriver configuration value=] of
   [=scrollbar type override configuration=] for |navigable|.

1. Assert: |scrollbar type override| is "<code>classic</code>", "<code>overlay</code>" or
   [=WebDriver configuration/unset=].

1. If |scrollbar type override| is "<code>classic</code>", run [=implementation-defined=] steps to
   make the |navigable|'s [=active document=] to use <a>classic scrollbars</a> and return.

1. If |scrollbar type override| is "<code>overlay</code>", run [=implementation-defined=]
   steps to make the |navigable|'s [=active document=] to use <a>overlay scrollbars</a> and return.

1. Assert: |scrollbar type override| is [=WebDriver configuration/unset=].

1. Run [=implementation-defined=] steps to make the |navigable|'s [=active document=] to
   use an [=implementation-defined=] default scrollbar type.

</div>

<div algorithm="remote end steps for emulation.setScrollbarTypeOverride">

The [=remote end steps=] given |command parameters| are:

1. Let |scrollbar type override| be |command parameters|["<code>scrollbarType</code>"].

1. If |scrollbar type override| is null, set |scrollbar type override| to
   [=WebDriver configuration/unset=].

1. If the implementation does not support setting |scrollbar type override|, then return [=error=]
   with [=error code=] [=unsupported operation=].

1. Let |affected navigables| be the result of [=trying=] to [=store WebDriver configuration=]
   [=scrollbar type override configuration=] |scrollbar type override| for |command parameters|.

1. For each |navigable| of |affected navigables|:

   1. [=Update scrollbar type override=] for |navigable|.

1. Return [=success=] with data null.

</div>


#### The emulation.setTimezoneOverride Command ####  {#command-emulation-setTimezoneOverride}

The <dfn export for=commands>emulation.setTimezoneOverride</dfn> command modifies
timezone on the given top-level traversables or user contexts.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetTimezoneOverride = (
        method: "emulation.setTimezoneOverride",
        params: emulation.SetTimezoneOverrideParameters
      )

      emulation.SetTimezoneOverrideParameters = {
        timezone: text / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetTimezoneOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="updated SystemTimeZoneIdentifier steps">
The [=SystemTimeZoneIdentifier=] algorithm is implementation defined. A
WebDriver-BiDi [=remote end=] must have an implementation that runs the following
steps:

1. Let |emulated timezone| be null.

1. Let |realm| be [=current Realm Record=].

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |realm|.

1. Let |related navigables| be the result of [=get related navigables=] given
   |environment settings|.

1. For each |navigable| of |related navigables|:

    1. Let |top-level traversable| be |navigable|’s
       [=navigable/top-level traversable=].

    1. Let |user context| be |top-level traversable|'s [=associated user context=].

    1. If [=timezone overrides map=] [=map/contains=] |top-level traversable|, set
       |emulated timezone| to [=timezone overrides map=][|top-level traversable|].

    1. Otherwise, if [=timezone overrides map=] [=map/contains=]
       |user context|, set |emulated timezone| to
       [=timezone overrides map=][|user context|].

1. If |emulated timezone| is not null, return |emulated timezone|.

1. Return the result of implementation-defined steps in accordance with the
   requirements of the [=SystemTimeZoneIdentifier=] specification.

</div>

<div algorithm="remote end steps for emulation.setTimezoneOverride">

The [=remote end steps=] with |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. If |command parameters| doesn't [=map/contain=] "<code>userContexts</code>"
   and |command parameters| doesn't [=map/contain=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |emulated timezone| be |command parameters|["<code>timezone</code>"].

1. If |emulated timezone| is not null and
   [=IsTimeZoneOffsetString=](|emulated timezone|) returns false and
   [=AvailableNamedTimeZoneIdentifiers=] does not [=list/contain=]
   |emulated timezone|, return [=error=] with [=error code=] [=invalid argument=].

1. Let |navigables| be a [=/set=].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to
      [=get valid top-level traversables by ids=] with
      |command parameters|["<code>contexts</code>"].

1. Otherwise:

   1. Assert the <code>userContexts</code> field of |command parameters| is present.

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=]
      with |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |emulated timezone| is null, [=map/remove=] |user context| from
         [=timezone overrides map=].

      1. Otherwise, [=map/set=] [=timezone overrides map=][|user context|] to |emulated timezone|.

      1. [=list/For each=] |top-level traversable| of the list of all
         [=/top-level traversables=] whose [=associated user context=] is
         |user context|:

         1. [=list/Append=] |top-level traversable| to |navigables|.

1. For each |navigable| of |navigables|:

   1. If |emulated timezone| is null, [=map/remove=] |navigable| from [=timezone overrides map=].

   1. Otherwise, [=map/set=] [=timezone overrides map=][|navigable|] to |emulated timezone|.

1. Return [=success=] with data null.

</div>

#### The emulation.setTouchOverride Command ####  {#command-emulation-setTouchOverride}

The <dfn export for=commands>emulation.setTouchOverride</dfn> command emulates
enabled touch input on web pages.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      emulation.SetTouchOverride = (
        method: "emulation.setTouchOverride",
        params: emulation.SetTouchOverrideParameters
      )

      emulation.SetTouchOverrideParameters = {
        maxTouchPoints: (js-uint .ge 1) / null,
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      emulation.SetTouchOverrideResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
The <dfn export>WebDriver BiDi emulated max touch points</dfn> steps given
[=environment settings object=] |environment settings| are:

1. Let |related navigables| be the result of [=get related navigables=] with |environment settings|.

1. For each |navigable| of |related navigables|:

   1. Let |top-level navigable| be |navigable|'s [=navigable/top-level traversable=].

   1. Let |user context| be |top-level navigable|'s [=associated user context=].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/navigables=] contains |top-level navigable|, return
         |session|'s [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/navigables=][|top-level navigable|].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/user contexts=] contains |user context|, return |session|'s
         [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/user contexts=][|user context|].

1. For each |session| in [=active BiDi sessions=]:

   1. Let |emulated maxTouchPoints| be |session|'s [=session/emulated maxTouchPoints=]'s
      [=emulated maxTouchPoints/default=].

   1. If |emulated maxTouchPoints| is not null, return |emulated maxTouchPoints|.

1. Return null.

</div>

<div algorithm="remote end steps for emulation.setTouchOverride">

The [=remote end steps=] with |session| and |command parameters| are:

Note: There is a legacy [=expose legacy touch event APIs=], which can still be used by some existing
web contents as a signal that the user agent is a touch-enabled "mobile" device. Even though the API
is legacy, user agent might run [=implementation-defined=] steps to respect the
[=session/emulated maxTouchPoints=] state in the [=expose legacy touch event APIs=].

1. If |command parameters| [=map/contains=] "<code>userContexts</code>" and |command parameters|
   [=map/contains=] "<code>contexts</code>", return [=error=] with [=error code=]
   [=invalid argument=].

1. Let |maxTouchPoints| be |command parameters|["<code>maxTouchPoints</code>"].

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Let |navigables| be the result of [=trying=] to [=get valid top-level traversables by ids=]
      with |command parameters|["<code>contexts</code>"].

   1. For each |navigable| of |navigables|:

      1. If |maxTouchPoints| is null, [=map/remove=] |navigable| from |session|'s
         [=session/emulated maxTouchPoints=]'s [=emulated maxTouchPoints/navigables=].

      1. Otherwise, [=map/set=] |session|'s [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/navigables=][|navigable|] to |maxTouchPoints|.

   1. Return [=success=] with data null.

1. If the <code>userContexts</code> field of |command parameters| is present:

   1. Let |user contexts| be the result of [=trying=] to [=get valid user contexts=] with
      |command parameters|["<code>userContexts</code>"].

   1. For each |user context| of |user contexts|:

      1. If |maxTouchPoints| is null, [=map/remove=] |user context| from |session|'s
         [=session/emulated maxTouchPoints=]'s [=emulated maxTouchPoints/user contexts=].

      1. Otherwise [=map/set=] |session|'s [=session/emulated maxTouchPoints=]'s
         [=emulated maxTouchPoints/user contexts=][|user context|] to |maxTouchPoints|.

   1. Return [=success=] with data null.

1. Set |session|'s [=session/emulated maxTouchPoints=]'s [=emulated maxTouchPoints/default=]
   to |maxTouchPoints|.

1. Return [=success=] with data null.

</div>

## The network Module ## {#module-network}

The <dfn export for=modules>network</dfn> module contains commands and events
relating to network requests.

### Definition ### {#module-network-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">

NetworkCommand = (
  network.AddDataCollector //
  network.AddIntercept //
  network.ContinueRequest //
  network.ContinueResponse //
  network.ContinueWithAuth //
  network.DisownData //
  network.FailRequest //
  network.GetData //
  network.ProvideResponse //
  network.RemoveDataCollector //
  network.RemoveIntercept //
  network.SetCacheBehavior //
  network.SetExtraHeaders
)

</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">

NetworkResult = (
  network.AddDataCollectorResult /
  network.AddInterceptResult /
  network.ContinueRequestResult /
  network.ContinueResponseResult /
  network.ContinueWithAuthResult /
  network.DisownDataResult /
  network.FailRequestResult /
  network.GetDataResult /
  network.ProvideResponseResult /
  network.RemoveDataCollectorResult /
  network.RemoveInterceptResult /
  network.SetCacheBehaviorResult /
  network.SetExtraHeadersResult
)

NetworkEvent = (
    network.AuthRequired //
    network.BeforeRequestSent //
    network.FetchError //
    network.ResponseCompleted //
    network.ResponseStarted
)

</pre>

A [=remote end=] has a <dfn>before request sent map</dfn> which is initially an
empty map.  It's used to track the network events for which a
<code>network.beforeRequestSent</code> event has already been sent.

A [=remote end=] has a <dfn>default cache behavior</dfn> which is a string. It is
initially "<code>default</code>".

A [=remote end=] has a <dfn>navigable cache behavior map</dfn> which is a weak
map between [=/top-level traversables=] and strings representing cache
behavior. It is initially empty.

A [=BiDi session=] has a <dfn for=session>extra headers</dfn> which is a
[=struct=] with an [=struct/item=] named <dfn for="extra headers">default
headers</dfn>, which is a [=/header list=] (initially set to an empty
[=/header list=]), an [=struct/item=] named
<dfn for="extra headers">user context headers</dfn>, which is a weak map
between [=user context|user contexts=] and [=/header lists=], and  a [=struct/item=] named
<dfn for="extra headers">navigable headers</dfn>, which is a weak map
between [=/navigables=] and [=/header lists=].

### Network Data Collection ### {#network-data-collection}

A <dfn>network data</dfn> is a [=/struct=] with:
* [=struct/Item=] named <dfn for="network-data">bytes</dfn>, which is a <code>[=network.BytesValue=]</code> or null,
* [=struct/Item=] named <dfn for="network-data">cloned body</dfn>, which is a [=/body=] or null,
* [=struct/Item=] named <dfn for="network-data">collectors</dfn>, which is a list of <code>[=network.Collector=]</code>,
* [=struct/Item=] named <dfn for="network-data">pending</dfn>, which is a boolean,
* [=struct/Item=] named <dfn for="network-data">request</dfn>, which is a [=request id=],
* [=struct/Item=] named <dfn for="network-data">size</dfn>, which is a js-uint or null,
* [=struct/Item=] named <dfn for="network-data">type</dfn>, which is a <code>[=network.DataType=]</code>.

A <dfn for="network">collector</dfn> is a [=/struct=] with:
* [=struct/Item=] named <dfn for="network-collector">max encoded item size</dfn>, which is a js-uint;
* [=struct/item=] named <dfn for="network-collector">contexts</dfn>, which is a [=/list=] of [=navigable id=];
* [=struct/item=] named <dfn for="network-collector">data types</dfn>, which is a [=/list=] of  <code>[=network.DataType=]</code>;
* [=struct/item=] named <dfn for="network-collector">collector</dfn>, which is a <code>[=network.Collector=]</code>;
* [=struct/item=] named <dfn for="network-collector">collector type</dfn>, which is a <code>[=network.CollectorType=]</code>;
* [=struct/item=] named <dfn for="network-collector">user contexts</dfn>, which is a [=/list=] of <code>[=browser.UserContext=]</code>.

Note: [=network-collector/max encoded item size=] defines the limit per item (response or request),
and does not limit the size collected by the specific collector. The total size of all collected resources is
limited by [=max total collected size=].

A [=BiDi session=] has <dfn>network collectors</dfn> which is a [=/map=] between
<code>[=network.Collector=]</code> and a [=network/collector=]. It is initially empty.

A [=remote end=] has <dfn>collected network data</dfn> which is a list of
[=network data=]. It is initially empty.

A [=remote end=] has a <dfn>max total collected size</dfn> which is a js-uint representing
the size allocated to collect network data in [=collected network data=]. Its
value is implementation-defined.

Note: This allows implementations to set resource usage limits.
It is expected that the limits are sufficiently large that users can depend on
collecting data that is fully decoded and handled by the browser,
such as images and fonts used on a webpage.

<div algorithm>
To <dfn>get navigable for request</dfn> given request:

1. Let |navigable| be null.

1. If |request|'s [=request/client=] is an [=environment settings object=]:

   1. Let |environment settings| be |request|'s [=request/client=]

   1. If there is a [=/navigable=] whose [=active window=] is |environment
      settings|' [=environment settings object/global object=], set |navigable|
      to be that navigable.

1. Return |navigable|.

</div>

<div algorithm>
To <dfn>match collector for navigable</dfn> given |collector| and |navigable|:

1. If |collector|'s [=network-collector/contexts=] is not [=list/empty=]:

  1. If |collector|'s [=network-collector/contexts=] [=list/contains=] |navigable|'s [=navigable id=], return true.

  1. Otherwise, return false.

1. If |collector|'s [=network-collector/user contexts=] is not [=list/empty=]:

  1. Let |user context| be |navigable|'s [=associated user context=].

  1. If |collector|'s [=network-collector/user contexts=] [=list/contains=] |user context|'s
     [=user context id=], return true.

  1. Otherwise, return false.

1. Return true.

</div>

<div algorithm>
The <dfn export>WebDriver BiDi clone network request body</dfn> steps given [=/request=] |request| are:

1. If |request|'s [=request/body=] is null, return.

1. For each |session| in [=active BiDi sessions=]:

   1. If |session|'s [=network collectors=] is not [=list/empty=]:

      1. Let |collected data| be a [=network data=] with
         [=network-data/bytes=] set to null,
         [=network-data/cloned body=] set to [=body/clone=] of |request|'s [=request/body=],
         [=network-data/collectors=] set to an empty list,
         [=network-data/pending=] set to true,
         [=network-data/request=] set to |request|'s [=request id=],
         [=network-data/size=] set to null,
         [=network-data/type=] set to "request".

      1. [=list/Append=] |collected data| to [=collected network data=].

      1. Return.

</div>

<div algorithm>
The <dfn export>WebDriver BiDi clone network response body</dfn> steps given |request| and |response body| are:

1. If |response body| is null, return.

1. For each |session| in [=active BiDi sessions=]:

   1. If |session|'s [=network collectors=] is not [=list/empty=]:

      1. Let |collected data| be a [=network data=] with
         [=network-data/bytes=] set to null,
         [=network-data/cloned body=] set to [=body/clone=] of |response body|,
         [=network-data/collectors=] set to an empty list,
         [=network-data/pending=] set to true,
         [=network-data/request=] set to |request|'s [=request id=],
         [=network-data/size=] set to null,
         [=network-data/type=] set to "response".

      1. [=list/Append=] |collected data| to [=collected network data=].

      1. Return.

</div>

<div algorithm>
To <dfn>get collected data</dfn> given |request id| and |data type|.

1. For |collected data| of [=collected network data=]:

   1. If |collected data|'s [=network-data/request=] is |request id| and
      |collected data|'s [=network-data/type=] is |data type|, return |collected data|.

1. Return null.

</div>

<div algorithm>
To <dfn>maybe abort network response body collection</dfn> given |request|:

1. Let |collected data| be [=get collected data=] with |request|'s [=request id=] and "response".

1. If |collected data| is null, return.

1. Set |collected data|'s [=network-data/pending=] to false.

1. [=Resume=] with "<code>network data collected</code>" and (|request|'s [=request id=], "response").

</div>

<div algorithm>
To <dfn>maybe collect network request body</dfn> given |request|:

1. Let |collected data| be [=get collected data=] with |request|'s [=request id=] and "request".

1. If |collected data| is null, return.

   NOTE: This might happen if there are no collectors setup when the request is created,
   and [=WebDriver BiDi clone network request body=] does not clone the corresponding body.
   Or if the body was null in the first place.

1. [=Maybe collect network data=] with |request|, |collected data|, null and "request".

</div>

<div algorithm>
To <dfn>maybe collect network response body</dfn> given |request| and |response|:

1. If |response|'s [=response/status=] is a [=redirect status=], return.

   NOTE: For redirects, only the final response body is stored.

1. Let |collected data| be [=get collected data=] with |request|'s [=request id=] and "response".

1. If |collected data| is null, return.

   NOTE: This might happen if there are no collectors setup when the response is created,
   and [=WebDriver BiDi clone network response body=] does not clone the corresponding body.
   Or if the body was null in the first place.

1. Let |size| be |response|'s [=response body info=]'s [=encoded size=].

   NOTE: There is a discrepancy between the fact that the bytes retrieved from the
   fetch stream correspond to the decoded data, but the encoded (network) size is
   used in order to calculate size limits. Implementations might decide to use a storage
   model such that it uses less size than storing the decoded data, as long as the data
   returned to clients in getData is identical to the decoded data. The potential
   tradeoff between storage and performance is up to the implementation.

1. [=Maybe collect network data=] with |request|, |collected data|, |size| and "response".

</div>

<div algorithm>
To <dfn>maybe collect network data</dfn> given [=/request=] |request|,
[=network data=] |collected data|, js-uint |size| and [=network.DataType=] |data type|:

1. Set |collected data|'s [=network-data/pending=] to false.

1. Let |navigable| be [=get navigable for request=] with |request|.

1. If |navigable| is null:

   1. [=list/Remove=] |collected data| from [=collected network data=].

   1. [=Resume=] with "<code>network data collected</code>" and (|request|'s [=request id=], |data type|).

   1. Return.

   ISSUE: This prevents collecting data not related to a navigable.
   We still need to retrieve the navigable to check against the collector
   configuration but we could still accept null here.

1. Let |top-level navigable| be |navigable|'s [=navigable/top-level traversable=].

1. Let |collectors| be an empty list.

1. For each |session| in [=active BiDi sessions=]:

   1. For each |collector| in |session|'s [=network collectors=]:

      1. If |collector|'s [=network-collector/data types=] [=list/contains=] |data type| and if
         [=match collector for navigable=] with |collector| and |top-level navigable|:

         1. [=list/Append=] |collector| to |collectors|.

1. If |collectors| is [=list/empty=]:

   1. [=list/Remove=] |collected data| from [=collected network data=].

   1. [=Resume=] with "<code>network data collected</code>" and (|request|'s [=request id=], |data type|).

   1. Return.

1. Let |bytes| be null.

1. Let |processBody| given |nullOrBytes| be this step:

   1. If |nullOrBytes| is not null:

      1. Set |bytes| to [=serialize protocol bytes=] with |nullOrBytes|.

      1. If |size| is null, set |size| to |bytes|' [=byte sequence/length=].

1. Let |processBodyError| be this step: Do nothing.

1. [=Fully read=] |collected data|’s [=network-data/cloned body=] given |processBody| and |processBodyError|.

1. If |bytes| is not null:

   1. For |collector| in |collectors|:

      1. If |size| is less than or equal to |collector|'s [=network-collector/max encoded item size=],
         [=list/append=] |collector|'s [=network-collector/collector=] to |collected data|'s
         [=network-data/collectors=].

   1. If |collected data|'s [=network-data/collectors=] is not [=list/empty=]:

      1. [=Allocate size to record data=] given |size|.

      1. Set |collected data|'s [=network-data/bytes=] to |bytes|.

      1. Set |collected data|'s [=network-data/size=] to |size|.

   1. Otherwise, [=list/remove=] |collected data| from [=collected network data=].

1. [=Resume=] with "<code>network data collected</code>" and (|request|'s [=request id=], |data type|).

</div>

<div algorithm>
To <dfn>allocate size to record data</dfn> given |size|:

1. Let |available size| be [=max total collected size=].

1. Let |already collected data| be an empty list.

1. For each |collected data| in [=collected network data=]:

   1. If |collected data|'s [=network-data/bytes=] is not null:

      1. Decrease |available size| by |collected data|'s [=network-data/size=].

      1. [=list/Append=] |collected data| to |already collected data|

1. If |size| is greater than |available size|:

   1. For each |collected data| in |already collected data|:

      1. Increase |available size| by |collected data|'s [=network-data/size=].

      1. Set |collected data|'s [=network-data/bytes=] field to null.

      1. Set |collected data|'s [=network-data/size=] field to null.

      1. If |available size| is greater than or equal to |size|, return.

</div>

<div algorithm>
To <dfn>remove collector from data</dfn> given |collected data| and |collector id|:

1. If |collected data|'s [=network-data/collectors=] [=list/contains=] |collector id|:

   1. [=list/Remove=] |collector id| from |collected data|'s [=network-data/collectors=].

   1. If |collected data|'s [=network-data/collectors=] is [=list/empty=]:

      1. [=list/Remove=] |collected data| from [=collected network data=].

</div>

### Network Intercepts ### {#network-intercepts}

A <dfn>network intercept</dfn> is a mechanism to allow remote ends to intercept
and modify network requests and responses.

A [=BiDi session=] has an <dfn>intercept map</dfn> which is a [=/map=] between
intercept id and a [=struct=] with fields <code>url patterns</code>,
<code>phases</code>, and <code>contexts</code> that define the properties
of active network intercepts. It is initially empty.

A [=BiDi session=] has a <dfn>blocked request map</dfn>, used to track the
requests which are actively being blocked. It is an [=/map=] between [=request id=]
and a [=struct=] with fields <code>request</code>, <code>phase</code>, and
<code>response</code>. It is initially empty.

<div algorithm>

To <dfn>get the network intercepts</dfn> given |session|, |event|, |request|, and |navigable id|:

1. Let |session intercepts| be |session|'s [=intercept map=].

1. Let |intercepts| be an empty list.

1. Run the steps under the first matching condition:
  <dl>
    <dt>|event| is "<code>network.beforeRequestSent</code>"
    <dd>Set |phase| to "<code>beforeRequestSent</code>".
    <dt>|event| is "<code>network.responseStarted</code>"
    <dd>Set |phase| to "<code>responseStarted</code>".
    <dt>|event| is "<code>network.authRequired</code>"
    <dd>Set |phase| to "<code>authRequired</code>".
    <dt>|event| is "<code>network.responseCompleted</code>"
    <dd>Return |intercepts|.
  </dl>

1. Let |url| be the result of running the [=URL serializer=] with |request|'s
   [=request/URL=].

1. For each |intercept id| → |intercept| of |session intercepts|:

  1. If |intercept|'s <code>contexts</code> is not null:

    1. If |intercept|'s <code>contexts</code> does not [=set/contains|contain=] |navigable id|:

      1. Continue.

  1. If |intercept|'s <code>phases</code> [=set/contains=] |phase|:

    1. Let |url patterns| be |intercept|'s <code>url patterns</code>.

    1. If |url patterns| is [=list/empty=]:

      1. [=list/Append=] |intercept id| to |intercepts|.

      1. Continue.

    1. For each |url pattern| in |url patterns|:

      1. If [=match URL pattern=] with |url pattern| and |url|:

        1. [=list/Append=] |intercept id| to |intercepts|.

        1. Break.

1. Return |intercepts|.

</div>

<div algorithm>
To <dfn>update the response</dfn> given |session|, |command| and |command parameters|:

1. Let |blocked requests| be |session|'s [=blocked request map=].

1. Let |request id| be |command parameters|["<code>request</code>"].

1. If |blocked requests| does not [=map/contain=] |request id| then return
   [=error=] with [=error code=] [=no such request=].

1. Let (<var ignore>request</var>, |phase|, |response|) be |blocked requests|[|request id|].

1. If |phase| is "<code>beforeRequestSent</code>" and |command| is
   "<code>continueResponse</code>", return [=error=] with [=error code=]
   "<code>invalid argument</code>".

    TODO: Consider a different error

1. If |response| is null:

  <!-- We can end up with non-null response for beforeRequestSent if we have
  multiple sessions that all intercept the same request -->
  1. Assert: |phase| is "<code>beforeRequestSent</code>".

  1. Set |response| to a new [=/response=].

1. If |command parameters| [=map/contains=] "<code>statusCode</code>":

  1. Set |responses|'s [=response/status=] be |command
     parameters|["<code>statusCode</code>"].

1. If |command parameters| [=map/contains=] "<code>reasonPhrase</code>":

  1. Set |responses|'s [=response/status message=] be [=UTF-8 encode=] with
     |command parameters|["<code>reasonPhrase</code>"].

1. If |command parameters| [=map/contains=] "<code>headers</code>":

  1. Let |headers| be the result of [=trying=] to [=create a headers list=] with
     |command parameters|["<code>headers</code>"].

  1. Set |response|'s [=response/header list=] to |headers|.

1. If |command parameters| [=map/contains=] "<code>cookies</code>":

  1. If |command parameters| [=map/contains=] "<code>headers</code>", let
     |headers| be |response|'s [=response/header list=].

     Otherwise:

     1. Let |headers| be an empty [=/header list=].

     1. For each |header| in |response|'s [=response/headers list=]:

       1. Let |name| be |header|'s name.

       1. If [=byte-lowercase=] |name| is not `<code>set-cookie</code>`:

         1. Append |header| to |headers|

  1. For |cookie| in |command parameters|["<code>cookies</code>"]:

    1. Let |header value| be [=serialize set-cookie header=] with |cookie|.

    1. Append (`<code>Set-Cookie</code>`, |header value|) to |headers|.

    1. Set |response|'s [=response/header list=] to |headers|.

1. If |command parameters| [=map/contains=] "<code>credentials</code>":

   Issue: This doesn't have a way to cancel the auth.

   1. Let |credentials| be |command parameters|["<code>credentials</code>"].

   1. Assert: |credentials|["<code>type</code>"] is "<code>password</code>".

   1. Set |response|'s authentication credentials <!--TODO: link--> to
      (|credentials|["<code>username</code>"], |credentials|["<code>password</code>"])

1. Return |response|

</div>

### Types ### {#module-network-types}

#### The network.AuthChallenge Type #### {#type-network-AuthChallenge}

<pre class="cddl" data-cddl-module="local-cddl">
network.AuthChallenge = {
  scheme: text,
  realm: text,
}
</pre>

<div algorithm>

To <dfn>extract challenges</dfn> given |response|:

Issue: Should we include parameters other than realm?

1. If |response|'s [=response/status=] is 401, let |header name| be
   `<code>WWW-Authenticate</code>`. Otherwise if |response|'s [=response/status=] is 407, let
   |header name| be `<code>Proxy-Authenticate</code>`. Otherwise return null.

1. Let |challenges| be a new [=/list=].

1. For each (|name|, |value|) in |response|'s [=response/header list=]:

   Issue: as in Fetch it's unclear if this is the right way to handle multiple
   headers, parsing issues, etc.

  1. If |name| is a [=byte-case-insensitive=] match for |header name|:

    1. Let |header challenges| be the result of parsing |value| into a list of
       challenges, each consisting of a scheme and a list of parameters, each of
       which is a [=tuple=] (name, value), according to the rules of [[!RFC9110]].

    1. For each |header challenge| in |header challenges|:

      1. Let |scheme| be |header challenge|'s scheme.

      1. Let |realm| be the empty string.

      1. For each (|param name|, |param value|) in |header challenge|'s
         parameters:
            1. If |param name| equals `<code>realm</code>` let |realm| be
               [=UTF-8 decode=] |param value|.

      1. Let |challenge| be a new [=/map=] matching the
         <code>network.AuthChallenge</code> production, with the
         <code>scheme</code> field set to |scheme| and the <code>realm</code>
         field set to |realm|.

     1. [=list/Append=] |challenge| to |challenges|.

1. Return |challenges|.

</div>

#### The network.AuthCredentials Type #### {#type-network-AuthCredentials}

<pre class="cddl" data-cddl-module="remote-cddl">
network.AuthCredentials = {
  type: "password",
  username: text,
  password: text,
}
</pre>

The <code>network.AuthCredentials</code> type represents the response to a
request for authorization credentials.

#### The network.BaseParameters Type #### {#type-network-BaseParameters}

<pre class="cddl" data-cddl-module="local-cddl">
network.BaseParameters = (
    context: browsingContext.BrowsingContext / null,
    isBlocked: bool,
    navigation: browsingContext.Navigation / null,
    redirectCount: js-uint,
    request: network.RequestData,
    timestamp: js-uint,
    ? userContext: browser.UserContext / null,
    ? intercepts: [+network.Intercept]
)
</pre>

The <code>network.BaseParameters</code> type is an abstract type representing
the data that's common to all network events.

Issue: Consider including the `sharedId` of the document node that initiated the
request in addition to the context.

<div algorithm>
To <dfn>process a network event</dfn> given |session|, |event|, and |request|:

1. Let |request data| be the result of [=get the request data=] with |request|.

<!-- TODO: update this to "[=request/navigation id=] once the fetch parts land-->
1. Let |navigation| be |request|'s [=navigation id=].

1. Let |navigable id| be null.

1. Let |top-level navigable id| be null.

1. Let |user context id| be null.

1. If |request|'s [=request/client=] is an [=environment settings object=]:

  1. Let |environment settings| be |request|'s [=request/client=].

  1. If there is a [=/navigable=] whose [=active window=] is |environment
     settings|' [=environment settings object/global object=], set |navigable id|
     to that navigable's [=navigable id=], set |top-level navigable id| to
     that navigable's [=navigable/top-level traversable=]'s [=navigable id=], and
     set |user context id| to the [=user context id=] of that navigable's
     [=associated user context=].

1. Let |intercepts| be the result of [=get the network intercepts=] with
   |session|, |event|, |request|, and |top-level navigable id|.

1. Let |redirect count| be |request|'s [=redirect count=].

1. Let |timestamp| be a [=time value=] representing the current date and time in UTC.

1. If |intercepts| is not [=list/empty=], let |is blocked| be true, otherwise
   let |is blocked| be false.

1. Let |params| be [=/map=] matching the <code>network.BaseParameters</code>
   production, with the <code>request</code> field set to |request data|, the
   |navigation| field set to <code>navigation</code>, the <code>context</code>
   field set to |navigable id|, the <code>userContext</code> field set to
   |user context id|, the <code>timestamp</code> field set to |timestamp|,
   the <code>redirectCount</code> field set to |redirect count|, the
   <code>isBlocked</code> field set to |is blocked|, and <code>intercepts</code>
   field set to |intercepts| if |is blocked| is true, or omitted otherwise.

1. Return |params|

</div>

#### The network.BytesValue Type #### {#type-network-BytesValue}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.BytesValue = network.StringValue / network.Base64Value;

network.StringValue = {
  type: "string",
  value: text,
}

network.Base64Value = {
  type: "base64",
  value: text,
}
</pre>

The <code><dfn>network.BytesValue</dfn></code> type represents binary data sent over the
network. Valid UTF-8 is represented with the <code>network.StringValue</code>
type, any other data is represented in Base64-encoded form as
<code>network.Base64Value</code>.

<div algorithm>
To <dfn>deserialize protocol bytes</dfn> given |protocol bytes|:

Note: this takes bytes encoded as a <code>[=network.BytesValue=]</code> and returns
a [=byte sequence=].

1. If |protocol bytes| matches the <code>network.StringValue</code> production,
   let |bytes| be [=UTF-8 encode=] |protocol bytes|["<code>value</code>"].

1. Otherwise if |protocol bytes| matches the <code>network.Base64Value</code>
   production. Let |bytes| be [=forgiving-base64 decode=] |protocol
   bytes|["<code>value</code>"].

1. Return |bytes|.

</div>

<div algorithm>
To <dfn>serialize protocol bytes</dfn> given |bytes|:

Note: this takes a [=byte sequence=] and returns a <code>[=network.BytesValue=]</code>.

1. Let |text| be [=UTF-8 decode without BOM or fail=] |bytes|.

1. If |text| is failure, return a [=/map=] matching the
   <code>network.Base64Value</code> production, with |value| set to
   [=forgiving-base64 encode=] |bytes|.

1. Return a [=/map=] matching the <code>network.StringValue</code> production,
   with |value| set to |text|.

</div>

#### The network.Collector Type #### {#type-network-Collector}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.Collector = text
</pre>

The <code><dfn>network.Collector</dfn></code> type represents the id of a [=network/collector=].

#### The network.CollectorType Type #### {#type-network-CollectorType}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.CollectorType = "blob"
</pre>

Note: In the future we might also support the "stream" collector type for clients
which want to read the data gathered by a given collector via a stream.

The <code><dfn>network.CollectorType</dfn></code> type represents the different types of data collectors
that can be added.

#### The network.Cookie Type #### {#type-network-Cookie}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">

network.SameSite = "strict" / "lax" / "none" / "default"

<!--
Modifications to this definition should be reflected in
`network.SetCookieHeader`, `storage.CookieFilter`, and `storage.PartialCookie`.
-->
network.Cookie = {
    name: text,
    value: network.BytesValue,
    domain: text,
    path: text,
    size: js-uint,
    httpOnly: bool,
    secure: bool,
    sameSite: network.SameSite,
    ? expiry: js-uint,
    Extensible,
}
</pre>

The <code>network.Cookie</code> type represents a cookie.

<div algorithm>
To <dfn>serialize cookie</dfn> given |stored cookie|:

1. Let |name| be the result of [=UTF-8 decode=] with |stored cookie|'s name field.

1. Let |value| be [=serialize protocol bytes=] with |stored cookie|'s value.

1. Let |domain| be |stored cookie|'s domain field.

1. Let |path| be |stored cookie|'s path field.

1. Let |expiry| be |stored cookie|'s expiry-time field represented as a unix
   timestamp, if set, or null otherwise.

1. Let |size| be the byte length of the result of serializing |stored cookie|
   as it would be represented in a <code>Cookie</code> header.

1. Let |http only| be true if |stored cookie|'s http-only-flag is true, or false
   otherwise.

1. Let |secure| be true if |stored cookie|'s secure-only-flag is true, or false
   otherwise.

1. Let |same site| be "<code>none</code>" if |stored cookie|'s same-site-flag is
   "<code>None</code>", "<code>lax</code>" if it is "<code>Lax</code>",
   "<code>strict</code>" if it is "<code>Strict</code>", or
   "<code>default</code>" if it is "<code>Default</code>"

1. Return a map matching the <code>network.Cookie</code> production,
   with the <code>name</code> field set to |name|, the <code>value</code> field
   set to |value|, the <code>domain</code> field set to |domain|, the
   <code>path</code> field set to |path|, the <code>expiry</code> field set to
   |expiry| if it's not null, or omitted otherwise, the <code>size</code> field
   set to |size|, the <code>httpOnly</code> field set to |http only|, the
   <code>secure</code> field set to |secure|, and the <code>sameSite</code>
   field set to |same site|.

</div>

#### The network.CookieHeader Type #### {#type-network-CookieHeader}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
network.CookieHeader = {
    name: text,
    value: network.BytesValue,
}
</pre>

The <code>network.CookieHeader</code> type represents the subset of cookie data
that's in a <code>Cookie</code> request header.

<div algorithm>
To <dfn>serialize cookie header</dfn> given |protocol cookie|:

1. Let |name| be [=UTF-8 encode=] |protocol cookie|["<code>name</code>"].

1. Let |value| be [=deserialize protocol bytes=] with |protocol
   cookie|["<code>value</code>"].

1. Let |header value| be the [=byte sequence=] formed by concatenating |name|,
   `<code>=</code>`, and |value|

1. Return |header value|.

</div>

#### The network.DataType Type #### {#type-network-dataType}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.DataType = "request" / "response"
</pre>

The <code><dfn>network.DataType</dfn></code> type represents the different types of network data
that can be collected.

#### The network.FetchTimingInfo Type #### {#type-network-FetchTimingInfo}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
network.FetchTimingInfo = {
    timeOrigin: float,
    requestTime: float,
    redirectStart: float,
    redirectEnd: float,
    fetchStart: float,
    dnsStart: float,
    dnsEnd: float,
    connectStart: float,
    connectEnd: float,
    tlsStart: float,
    <!-- tlsEnd: float this should be the same as connectEnd -->
    requestStart: float,
    responseStart: float,
    <!-- TODO responseHeadersEnd: float: Not sure quite what to use for this -->
    responseEnd: float,
}
</pre>

The <code>network.FetchTimingInfo</code> type represents the time of each part
of the request, relative to the <a spec=html>time origin</a> of the [=/request=]'s
[=request/client=].

<div algorithm>
To <dfn>get the fetch timings</dfn> given |request|:

1. Let |global| be |request|'s [=request/client=].

1. If |global| is null, return a map matching the
   <code>network.FetchTimingInfo</code> production, with all fields set to 0.

1. Let |time origin| be [=get time origin timestamp=] with |global|.

1. Let |timings| be |request|'s [=fetch timing info=].

1. Let |connection timing| be |timings|' [=final connection timing info=] if
   it's not null, or a new [=connection timing info=] otherwise.

1. Let |request time| be [=convert fetch timestamp=] given |timings|' <a spec=fetch>start time</a> and |global|.

1. Let |redirect start| be [=convert fetch timestamp=] given |timings|'
   [=redirect start time=] and |global|.

1. Let |redirect end| be [=convert fetch timestamp=] given |timings|' [=redirect
   end time=] and |global|.

1. Let |fetch start| be [=convert fetch timestamp=] given |timings|'
   [=post-redirect start time=] and |global|.

1. Let |DNS start| be [=convert fetch timestamp=] given |connection timing|'s
   [=domain lookup start time=] and |global|.

1. Let |DNS end| be [=convert fetch timestamp=] given |connection timing|'s
   [=domain lookup end time=] and |global|.

1. Let |TLS start| be [=convert fetch timestamp=] given |connection timing|'s
   [=secure connection start time=] and |global|.

1. Let |connect start| be [=convert fetch timestamp=] given |connection
   timing|'s [=connection start time=] and |global|.

1. Let |connect end| be [=convert fetch timestamp=] given |connection timing|'s
   [=connection end time=] and |global|.

1. Let |request start| be [=convert fetch timestamp=] given |timings|' [=final
   network-request start time=] and |global|.

1. Let |response start| be [=convert fetch timestamp=] given |timings|' [=final
   network-response start time=] and |global|.

1. Let |response end| be [=convert fetch timestamp=] given |timings|'
   <a spec=fetch>end time</a> and |global|.

1. Return a [=/map=] matching the <code>network.FetchTimingInfo</code> production
    with the <code>timeOrigin</code> field set to |time origin|, the
    <code>requestTime</code> field set to |request time|, the
    <code>redirectStart</code> field set to |redirect start|, the
    <code>redirectEnd</code> field set to |redirect end|, the
    <code>fetchStart</code> field set to |fetch start|, the
    <code>dnsStart</code> field set to |DNS start|, the <code>dnsEnd</code>
    field set to |DNS end|, the <code>connectStart</code> field set to |connect
    start|, the <code>connectEnd</code> field set to |connect end|, the
    <code>tlsStart</code> field set to |TLS start|, the
    <code>requestStart</code> field set to |request start|, the
    <code>responseStart</code> field set to |response start|, and the
    <code>responseEnd</code> field set to |response end|.

</div>

TODO: Add service worker fields

#### The network.Header Type #### {#type-network-Header}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.Header = {
  name: text,
  value: network.BytesValue,
}
</pre>

The <code>network.Header</code> type represents a single request header.

<div algorithm>
To <dfn>serialize header</dfn> given |name bytes| and |value bytes|:

1. Let |name| be the result of [=UTF-8 decode=] with |name bytes|.

   Assert: Since header names are constrained to be ASCII-only this cannot fail.

1. Let |value| be [=serialize protocol bytes=] with |value bytes|.

1. Return a map matching the <code>network.Header</code> production, with the
   <code>name</code> field set to |name|, and the <code>value</code> field
   set to |value|.

</div>

<div algorithm>
To <dfn>deserialize header</dfn> given |protocol header|:

1. Let |name| be [=UTF-8 encode=] |protocol header|["<code>name</code>"].

1. Let |value| be [=deserialize protocol bytes=] with |protocol
   header|["<code>value</code>"].

1. Return a [=/header=] (|name|, |value|).

</div>

<div algorithm>
To <dfn>create a headers list</dfn> given |protocol headers|:
1. Let |headers| be an empty [=/header list=].

1. For |header| in |protocol headers|:

  1. Let |deserialized header| be [=deserialize header=] with |header|.

  1. If |deserialized header|'s name does not match the [=field-name token=]
     production, return [=error=] with [=error code=]
    "<code>invalid argument</code>".

  1. If |deserialized header|'s value does not match the [=header value=]
     production, return [=error=] with [=error code=]
    "<code>invalid argument</code>".

  1. Append |deserialized header| to |headers|.

1. Return [=success=] with data |headers|

</div>

#### The network.Initiator Type #### {#type-network-Initiator}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
network.Initiator = {
    ? columnNumber: js-uint,
    ? lineNumber: js-uint,
    ? request: network.Request,
    ? stackTrace: script.StackTrace,
    ? type: "parser" / "script" / "preflight" / "other"
}
</pre>

The <code>network.Initiator</code> type represents the source of a network
request.

Note: The <code>type</code> field is included in the definition for backwards
compatibility, but is no longer set by the [=get the initiator=] steps, and will
be removed in a future revision of this specification. Its use is expected to be
replaced by <code>initiatorType</code> and <code>destination</code> on
<code>network.RequestData</code>.

Note: The <code>request</code> field is included in the definition for backwards
compatibility, but is no longer set by the [=get the initiator=] steps, and will
be removed in a future revision of this specification. The
<code>network.Initiator</code> is included in the
<code>network.BeforeRequestSentParameters</code> which also contain the same
request id, making this information redundant. See
[[#type-network-BaseParameters]].

<div algorithm>
To <dfn>get the initiator</dfn> given |request|:

1. If |request|'s [=request/initiator type=] is "<code>fetch</code>" or
   "<code>xmlhttprequest</code>":

  1. Let |stack trace| be the [=current stack trace=].

  1. If |stack trace| has size of 1 or greater, let |line number| be value of the
     <code>lineNumber</code> field in |stack trace|[0], and let |column number| be
     the value of the <code>columnNumber</code> field in |stack trace|[0]. Otherwise
     let |line number| and |column number| be 0.

  Otherwise, let |stack trace|, |column number|, and |line number| all be null.

  TODO: Chrome includes the current parser position as column number / line
  number for parser-inserted resources.

1. Return a [=/map=] matching the <code>network.Initiator</code> production, the
   <code>columnNumber</code> field set to |column number| if it's not null, or
   omitted otherwise, the <code>lineNumber</code> field set to |line number| if
   it's not null, or omitted otherwise and the <code>stackTrace</code> field set
   to |stack trace| if it's not null, or omitted otherwise.

</div>

#### The network.Intercept Type #### {#type-network-Intercept}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.Intercept = text
</pre>

The <code>network.Intercept</code> type represents the id of a [=network intercept=].

#### The network.Request Type #### {#type-network-Request}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
network.Request = text;
</pre>

Each network request has an associated <dfn export>request id</dfn>, which is a
string uniquely identifying that request. The identifier for a request resulting from a
redirect matches that of the request that initiated it.

#### The network.RequestData Type #### {#type-network-RequestData}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
network.RequestData = {
    request: network.Request,
    url: text,
    method: text,
    headers: [*network.Header],
    cookies: [*network.Cookie],
    headersSize: js-uint,
    bodySize: js-uint / null,
    destination: text,
    initiatorType: text / null,
    timings: network.FetchTimingInfo,
}
</pre>

The <code>network.RequestData</code> type represents an ongoing network request.

<div algorithm>

To <dfn>get the request data</dfn> given |request|:

1. Let |request id| be request's [=request id=].

1. Let |url| be the result of running the [=URL serializer=] with |request|'s
   [=request/URL=].

1. Let |method| be |request|'s [=request/method=].

1. Let |body size| be null.

1. Let |body| be request's [=request/body=].

1. If |body| is a [=byte sequence=], set |body size| to the length of that
   sequence. Otherwise, if |body| is a [=/body=] then set |body size| to that
   body's <a spec=fetch>length</a>.

1. Let |headers size| be the size in bytes of |request|'s [=request/headers list=] when
   serialized as mandated by [[HTTP11]].

   Note: For protocols which allow header compression, this is the compressed
   size of the headers, as sent over the network.

1. Let |headers| be an empty list.

1. Let |cookies| be an empty list.

1. For each (|name|, |value|) in |request|'s [=request/headers list=]:

  1. Append the result of [=serialize header=] with |name| and |value| to |headers|.

  1. If |name| is a [=byte-case-insensitive=] match for "<code>Cookie</code>" then:

    1. For each |cookie| in the user agent's [=cookie store=] that are included in
       |request|:

       Note: [[COOKIES]] defines some baseline requirements for which cookies in
       the store can be included in a request, but user agents are free to
       impose additional constraints.

       1. Append the result of [=serialize cookie=] given |cookie| to |cookies|.

1. Let |destination| be |request|'s [=request/destination=].

1. Let |initiator type| be |request|'s [=request/initiator type=].

1. Let |timings| be [=get the fetch timings=] with |request|.

1. Return a map matching the <code>network.RequestData</code> production, with
   the <code>request</code> field set to |request id|, <code>url</code> field
   set to |url|, the <code>method</code> field set to |method|, the
   <code>headers</code> field set to |headers|, the |cookies| field set to
   |cookies|, the <code>headersSize</code> field set to |headers size|, the
   <code>bodySize</code> field set to |body size|, the <code>destination</code>
   field set to |destination|, the <code>initiatorType</code> field set to
   |initiator type|, and the <code>timings</code> field set to |timings|.

</div>

#### The network.ResponseContent Type #### {#type-network-ResponseContent}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
network.ResponseContent = {
    size: js-uint
}
</pre>

The <code>network.ResponseContent</code> type represents the decoded response to
a network request.

<!-- Not sure this is worthwhile if it only ends up with a single field, but it
would be natural to add a field here if we have a way to return the body -->

<div algorithm>
To <dfn>get the response content info</dfn> given |response|.

1. Return a new map matching the <code>network.ResponseContent</code>
   production, with the <code>size</code> field set to |response|'s [=response body
   info=]'s [=decoded size=]

</div>

#### The network.ResponseData Type #### {#type-network-ResponseData}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
network.ResponseData = {
    url: text,
    protocol: text,
    status: js-uint,
    statusText: text,
    fromCache: bool,
    headers: [*network.Header],
    mimeType: text,
    bytesReceived: js-uint,
    headersSize: js-uint / null,
    bodySize: js-uint / null,
    content: network.ResponseContent,
    ?authChallenges: [*network.AuthChallenge],
}
</pre>

The <code>network.ResponseData</code> type represents the response to a network
request.

<div algorithm>

To <dfn>get the protocol</dfn> given |response|:

1. Let |protocol| be the empty string.

1. If |response|'s [=final connection timing info=] is not null, set |protocol|
   to |response|'s [=final connection timing info=]'s [=ALPN negotiated
   protocol=].

1. If |protocol| is the empty string, or is equal to "<code>unknown</code>":

  1. Set |protocol| to |response|'s [=response/url=]'s [=url/scheme=]

  1. If |protocol| is equal to either "<code>http</code>" or
     "<code>https</code>" and |response| has an associated HTTP Response.

     Note: [[FETCH]] isn't clear about the relation between a HTTP network response and
     a response object.

     <!-- It would be better to move this into Fetch itself. -->

    1. Let |http version| be the HTTP Response's Status line's HTTP-version [[HTTP11]].

    1. If |http version| starts with "<code>HTTP/</code>":

      1. Let |version| be the [=code unit substring=] of |http version| from 5
         to |http version|'s [=string/length=].

      1. If |version| is "<code>0.9</code>", set |protocol| to
         "<code>http/0.9</code>", otherwise if |version| is "<code>1.0</code>",
         set |protocol| to "<code>http/1.0</code>", otherwise if |version| is
         "<code>1.1</code>", set |protocol| to "<code>http/1.1</code>".

1. Return |protocol|.

</div>

<div algorithm>
To <dfn>get the response data</dfn> given |response|:

1. Let |url| be the result of running the [=URL serializer=] with |response|'s
   [=response/URL=].

1. Set |protocol| to [=get the protocol=] given |response|.

1. Let |status| be |response|'s [=response/status=].

1. Let |status text| be |response|'s [=status message=].

1. If |response|'s [=cache state=] is "<code>local</code>", let |from cache| be
   true, otherwise let it be false.

1. Let |headers| be an empty list.

1. Let |mime type| be the [=essence=] of the [=computed mime type=] for |response|.

   Note: this is whatever MIME type the browser is actually using, even if it
   isn't following the exact algorithm in the [[MIMESNIFF]] specification.

1. For each (|name|, |value|) in |response|'s [=response/headers list=]:

  1. Append the result of [=serialize header=] with |name| and |value| to |headers|.

1. Let |bytes received| be the total number of bytes transmitted as part of the
   HTTP response associated with |response|.

1. Let |headers size| be the number of bytes transmitted as part of the header
   fields section of the HTTP response.

1. Let |body size| be |response|'s [=response body info=]'s [=encoded size=].

1. Let |content| be the result of [=get the response content info=] with |response|.

1. Let |auth challenges| be the result of [=extract challenges=] with |response|.

1. Return a [=/map=] matching the <code>network.ResponseData</code> production,
   with the <code>url</code> field set to |url|, the <code>protocol</code> field
   set to |protocol|, the <code>status</code> field set to |status|, the
   <code>statusText</code> field set to |status text|, the
   <code>fromCache</code> field set to |from cache|, the <code>headers</code>
   field set to |headers|, the <code>mimeType</code> field set to |mime type|, the
   <code>bytesReceived</code> field set to |bytes received|, the
   <code>headersSize</code> field set to |headers size|, the
   <code>bodySize</code> field set to |body size|, <code>content</code>
   field set to |content|, and the <code>authChallenges</code> field set to
   |auth challenges| if it's not null, or omitted otherwise.

</div>

#### The network.SetCookieHeader Type #### {#type-network-SetCookieHeader}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
<!--
Modifications to this definition should be reflected in
`network.Cookie`, `storage.CookieFilter`, and `storage.PartialCookie`.
-->
network.SetCookieHeader = {
    name: text,
    value: network.BytesValue,
    ? domain: text,
    ? httpOnly: bool,
    ? expiry: text,
    ? maxAge: js-int,
    ? path: text,
    ? sameSite: network.SameSite,
    ? secure: bool,
}
</pre>

The <code>network.SetCookieHeader</code> represents the data in a
<code>Set-Cookie</code> response header.

<div algorithm>
<!-- TODO: move something like this into infra -->
To <dfn>serialize an integer</dfn> given |input| that is an integer:

Note: This produces the shortest representation of |input| as a string of
decimal digits.

1. Let |serialized| be an empty string.

1. Let |value| be |input|.

1. While |value| is greater than 0:

  1. Let |x| be |value| divided by 10.

  1. Let |most significant digits| be the integer part of |x|.

  1. Let |y| be |most significant digits| multiplied by 10.

  1. Let |least significant digit| be |value| - |y|.

  1. Assert: |least significant digit| is an integer in the range 0 to 9,
     inclusive.

  1. Let |codepoint| be the [=code point=] whose [=code point/value=] is U+0030
     DIGIT ZERO's [=code point/value=] + |least significant digit|.

  1. Prepend |codepoint| to |serialized|.

  1. Set |value| to |most significant digits|.

1. Return |serialized|.

</div>

<div algorithm>
To <dfn>serialize set-cookie header</dfn> given |protocol cookie|:

1. Let |name| be [=UTF-8 encode=] |protocol cookie|["<code>name</code>"].

1. Let |value| be [=deserialize protocol bytes=] with |protocol
   cookie|["<code>value</code>"].

1. Let |header value| be the [=byte sequence=] formed by concatenating |name|,
   `<code>=</code>`, and |value|.

1. If |protocol cookie| [=map/contains=] "<code>expiry</code>":

  1. Let |attribute| be `<code>;Expires=</code>`

  1. Append [=UTF-8 encode=] |protocol cookie|["<code>expiry</code>"] to
     |attribute|.

  1. Append |attribute| to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>maxAge</code>":

  1. Let |attribute| be `<code>;Max-Age=</code>`

  1. Let |max age string| be [=serialize an integer=] |protocol
    cookie|["<code>maxAge</code>"].

  1. Append [=UTF-8 encode=] |max age string| to |attribute|.

  1. Append |attribute| to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>domain</code>":

  1. Let |attribute| be `<code>;Domain=</code>`

  1. Append [=UTF-8 encode=] |protocol cookie|["<code>domain</code>"] to
     |attribute|.

  1. Append |attribute| to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>path</code>":

  1. Let |attribute| be `<code>;Path=</code>`

  1. Append [=UTF-8 encode=] |protocol cookie|["<code>path</code>"] to
     |attribute|.

  1. Append |attribute| to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>secure</code>" and |protocol
   cookie|["<code>secure</code>"] is true:

  1. Append `<code>;Secure</code>` to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>httpOnly</code>" and |protocol
   cookie|["<code>httpOnly</code>"] is true:

  1. Append `<code>;HttpOnly</code>` to |header value|.

1. If |protocol cookie| [=map/contains=] "<code>sameSite</code>":

  1. Let |attribute| be `<code>;SameSite=</code>`

  1. Append [=UTF-8 encode=] |protocol cookie|["<code>sameSite</code>"] to
     |attribute|.

  1. Append |attribute| to |header value|.

1. Return |header value|.

</div>

#### The network.UrlPattern Type #### {#type-network-UrlPattern}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
network.UrlPattern = (
  network.UrlPatternPattern /
  network.UrlPatternString
)

network.UrlPatternPattern = {
    type: "pattern",
    ?protocol: text,
    ?hostname: text,
    ?port: text,
    ?pathname: text,
    ?search: text,
}


network.UrlPatternString = {
    type: "string",
    pattern: text,
}

</pre>

A <code>network.UrlPattern</code> represents a pattern used for matching request
URLs for [=network intercepts=].

When URLs are matched against a <code>network.UrlPattern</code> the URL is
parsed, and each component is compared for equality with the corresponding field
in the pattern, if present. Missing fields from the pattern always match.

Note: This syntax is designed with future extensibility in mind. In particular
the syntax forbids characters that are treated specially in the [[URLPattern]]
specification. These can be escaped by prefixing them with a U+005C (\) character.

<div algorithm>
To <dfn>unescape URL pattern</dfn> given |pattern|

1. Let |forbidden characters| be the [=/set=] of codepoints «U+0028 ((), U+0029
   ()), U+002A (*), U+007B ({), U+007D (})»

1. Let |result| be the empty string.

1. Let |is escaped character| be false.

1. For each |codepoint| in |pattern|:

  1. If |is escaped character| is false:

    1. If |forbidden characters| [=set/contains=] |codepoint|, return [=error=]
       with [=error code=] [=invalid argument=].

    1. If |codepoint| is U+005C (\):

        1. Set |is escaped character| to true.

        1. Continue.

  1. Append |codepoint| to result.

  1. Set |is escaped character| to false.

1. Return [=success=] with data |result|.

</div>

<div algorithm>
To <dfn>parse URL pattern</dfn>, given |pattern|:

1. Let |has protocol| be true.

1. Let |has hostname| be true.

1. Let |has port| be true.

1. Let |has pathname| be true.

1. Let |has search| be true.

1. If |pattern| matches the <code>network.UrlPatternPattern</code> production:

  1. Let |pattern url| be the empty string.

  1. If |pattern| [=map/contains=] "<code>protocol</code>":

    1. If |pattern|["<code>protocol</code>"] is the empty string, return [=error=]
       with [=error code=] [=invalid argument=].

    1. Let |protocol| be the result of [=trying=] to [=unescape
       URL Pattern=] with |pattern|["<code>protocol</code>"].

    1. For each |codepoint| in |protocol|:

      1. If |codepoint| is not [=ASCII alphanumeric=] and «U+002B (+), U+002D
         (-), U+002E (.)» does not [=set/contains|contain=] |codepoint|:

        1. Return [=error=] with [=error code=] invalid argument.

    1. Append |protocol| to |pattern url|.

  1. Otherwise:

    1. Set |has protocol| to false.

    1. Append "<code>http</code>" to |pattern url|.

  1. Let |scheme| be [=ASCII lowercase=] with |pattern url|.

  1. Append "<code>:</code>" to |pattern url|.

  1. If |scheme| [=is special=], append "<code>//</code>" to |pattern url|.

  1. If |pattern| [=map/contains=] "<code>hostname</code>":

    1. If |pattern|["<code>hostname</code>"] is the empty string, return [=error=]
       with [=error code=] [=invalid argument=].

    1. If |scheme| is "<code>file</code>" return [=error=] with [=error code=]
       [=invalid argument=].

    1. Let |hostname| be the result of [=trying=] to [=unescape
       URL Pattern=] with |pattern|["<code>hostname</code>"].

    1. Let |inside brackets| be false.

    1. For each |codepoint| in |hostname|:

      1. If «U+002F (/), U+003F (?), U+0023 (#)» [=set/contains=]
         |codepoint|:

        1. Return [=error=] with [=error code=] [=invalid argument=].

      1. If |inside brackets| is false and |codepoint| is U+003A (:):

        1. Return [=error=] with [=error code=] [=invalid argument=].

      1. If |codepoint| is U+005B ([), set |inside brackets| to true.

      1. If |codepoint| is U+005D (]), set |inside brackets| to false.

    1. Append |hostname| to |pattern url|.

  1. Otherwise:

    1. If |scheme| is not "<code>file</code>", append "<code>placeholder</code>" to
       |pattern url|.

    1. Set |has hostname| to false.

  1. If |pattern| [=map/contains=] "<code>port</code>":

    1. If |pattern|["<code>port</code>"] is the empty string, return [=error=]
       with [=error code=] [=invalid argument=].

    1. Let |port| be the result of [=trying=] to [=unescape
       URL Pattern=] with |pattern|["<code>port</code>"].

    1. Append "<code>:</code>" to |pattern url|.

    1. For each |codepoint| in |port|:

      1. If |codepoint| is not an [=ASCII digit=]:

        1. Return [=error=] with [=error code=] [=invalid argument=].

    1. Append |port| to |pattern url|.

  1. Otherwise:

    1. Set |has port| to false.

  1. If |pattern| [=map/contains=] "<code>pathname</code>":

    1. Let |pathname| be the result of [=trying=] to [=unescape URL Pattern=] with
       |pattern|["<code>pathname</code>"].

    1. If |pathname| does not [=string/starts with|start with=] U+002F (/), then
       append "<code>/</code>" to |pattern url|.

    1. For each |codepoint| in |pathname|:

      1. If «U+003F (?), U+0023 (#)» [=set/contains=]
         |codepoint|:

        1. Return [=error=] with [=error code=] [=invalid argument=].

    1. Append |pathname| to |pattern url|.

  1. Otherwise:

    1. Set |has pathname| to false.

  1. If |pattern| [=map/contains=] "<code>search</code>":

    1. Let |search| be the result of [=trying=] to [=unescape URL pattern=] with
       |pattern|["<code>search</code>"].

    1. If |search| does not [=string/starts with|start with=] U+003F (?), then
       append "<code>?</code>" to |pattern url|.

    1. For each |codepoint| in |search|:

      1. If |codepoint| is U+0023 (#):

        1. Return [=error=] with [=error code=] [=invalid argument=].

    1. Append |search| to |pattern url|.

  1. Otherwise:

    1. Set |has search| to false.

1. Otherwise, if |pattern| matches the <code>network.UrlPatternString</code> production:

  1. Let |pattern url| be the result of [=trying=] to [=unescape URL
     pattern=] with |pattern|["<code>pattern</code>"].

1. Let |url| be the result of [=URL Parser|parsing=] |pattern url|.

1. If |url| is failure, return [=error=] with [=error code=] [=invalid argument=].

1. Let |parsed| be a struct with the following fields:
   <dl>
     <dt>protocol
     <dd>|url|'s [=url/scheme=] if |has protocol| is true, or null otherwise.
     <dt>hostname
     <dd>|url|'s [=url/host=] if |has hostname| is true, or null otherwise.
     <dt>port
     <dd>
     1. If |has port| is false:

       1. null.

     1. Otherwise:

       1. If |url|'s [=url/scheme=] [=is special=] and |url|'s [=url/scheme=]'s
          [=default port=] is not null, and |url|'s [=url/port=] is null or is
          equal to [=url/scheme=]'s [=default port=]:

         1. The empty string.

       1. Otherwise, if |url|'s [=url/port=] is not null:

         1. [=Serialize an integer=] with |url|'s [=url/port=].

       1. Otherwise:

         1. null.

     <dt>pathname
     <dd>
     1. If |has pathname| is false:

       1. null.

     1. Otherwise:

       1. The result of running the [=URL path serializer=] with |url|, if
         |url|'s [=url/path=] is not the empty string and is not [=list/empty=],
         or null otherwise.

     <dt>search
     <dd>
     1. If |has search| is false:

       1. null.

     1. Otherwise:

       1. The empty string if |url|'s [=url/query=] is null, or |url|'s [=url/query=] otherwise.
   </dl>

1. Return [=success=] with data |parsed|.

</div>

<div algorithm>
To <dfn>match URL pattern</dfn> given |url pattern| and |url string|:

1. Let |url| be the result of [=URL parser|parsing=] |url string|.

1. If |url pattern|'s protocol is not null and is not equal to |url|'s
   [=url/scheme=], return false.

1. If |url pattern|'s hostname is not null and is not equal to |url|'s
   [=url/host=], return false.

1. If |url pattern|'s port is not null:

   1. Let |port| be null.

   1. If |url|'s [=url/scheme=] [=is special=] and |url|'s
      [=url/scheme=]'s [=default port=] is not null, and |url|'s [=url/port=] is
      null or is equal to [=url/scheme=]'s [=default port=]:

     1. Set |port| to the empty string.

   1. Otherwise, if |url|'s [=url/port=], is not null:

     1. Set |port| to [=serialize an integer=] with |url|'s [=url/port=].

   1. If |url pattern|'s port is not equal to |port|, return false.

1. If |url pattern|'s pathname is not null and is not equal to the result of
   running the [=URL path serializer=] with |url|, return false.

1. If |url pattern|'s search is not null:

   1. Let |url query| be |url|'s [=url/query=].

   1. If |url query| is null, set |url query| to the empty string.

   1. If |url pattern|'s search is not equal to |url query|, return false.

1. Return true.

</div>

### Commands ### {#module-network-commands}

#### The network.addDataCollector Command ####  {#command-network-addDataCollector}

The <dfn export for=commands>network.addDataCollector</dfn> adds a
[=network/collector=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.AddDataCollector = (
        method: "network.addDataCollector",
        params: network.AddDataCollectorParameters
      )

      network.AddDataCollectorParameters = {
        dataTypes: [+network.DataType],
        maxEncodedDataSize: js-uint,
        ? collectorType: network.CollectorType .default "blob",
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      network.AddDataCollectorResult = {
        collector: network.Collector
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.addDataCollector">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |collector id| be the string representation of a [[!RFC9562|UUID]].

1. Let |input context ids| be an empty [=/set=].

1. If the <code>contexts</code> field of |command parameters| is present,
   set |input context ids| to [=set/create|create a set=] with
   |command parameters|[<code>contexts</code>].

1. Let |data types| be [=set/create|create a set=] with
   |command parameters|["<code>dataTypes</code>"].

1. Let |max encoded item size| be |command parameters|
   ["<code>maxEncodedDataSize</code>"].

   Note: The <code>maxEncodedDataSize</code> parameter represents
   [=network-collector/max encoded item size=] and limits the size of each request collected by the
   given collector, not the total collector's collected size.

   Note: Different implementations might support different encodings, which means the
   encoded size might be different between browsers. Therefore, for the same data collector
   configuration, some network data might fit the [=network-collector/max encoded item size=] only
   in some implementations.

1. Let |collector type| be |command parameters|
   ["<code>collectorType</code>"].

1. Let |input user context ids| be an empty [=/set=].

1. If the <code>userContexts</code> field of |command parameters| is present,
   set |input user context ids| to [=set/create|create a set=] with
   |command parameters|[<code>userContexts</code>].

1. If |input user context ids| is not empty and |input context ids| is not
   empty, return [=error=] with [=error code=] [=invalid argument=].

1. If |max encoded item size| is 0 or |max encoded item size| is greater than
   [=max total collected size=], return [=error=] with [=error code=] [=invalid argument=].

1. If |input context ids| is not [=set/empty=]:

   1. Let |navigables| be the result of [=trying=] to [=get valid navigables by ids=]
      with |input context ids|.

   1. For each |navigable| in |navigables|:

      1. If |navigable| is not a [=/top-level traversable=], return [=error=]
         with [=error code=] [=invalid argument=].

1. Otherwise, if |input user context ids| is not [=set/empty=]:

    1. [=list/For each=] |user context id| of |input user context ids|:

       1. Let |user context| be [=get user context=] with |user context id|.

       1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

1. Let |collector| be a [=network/collector=] with
   [=network-collector/max encoded item size=] field set to |max encoded item size|,
   [=network-collector/data types=] field set to |data types|,
   [=network-collector/collector=] field set to |collector id|,
   [=network-collector/collector type=] field set to |collector type|,
   [=network-collector/contexts=] field set to |input context ids|,
   [=network-collector/user contexts=] field set to |input user context ids|.

1. Set |session|'s [=network collectors=][|collector id|] to |collector|.

1. Return a new [=/map=] matching the
   <code>network.AddDataCollectorResult</code> production with the
   <code>collector</code> field set to |collector id|.

</div>

#### The network.addIntercept Command ####  {#command-network-addIntercept}

The <dfn export for=commands>network.addIntercept</dfn> command adds a
[=network intercept=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.AddIntercept = (
        method: "network.addIntercept",
        params: network.AddInterceptParameters
      )

      network.AddInterceptParameters = {
        phases: [+network.InterceptPhase],
        ? contexts: [+browsingContext.BrowsingContext],
        ? urlPatterns: [*network.UrlPattern],
      }

      network.InterceptPhase = "beforeRequestSent" / "responseStarted" /
                               "authRequired"
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      network.AddInterceptResult = {
        intercept: network.Intercept
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.addRequestIntercept">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |intercept| be the string representation of a [[!RFC9562|UUID]].

1. Let |url patterns| be the <code>urlPatterns</code> field of |command
   parameters| if present, or an empty [=/list=] otherwise.

1. Let |navigables| be null.

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Set |navigables| to an empty [=/set=].

   1. For each |navigable id| of |command parameters|["<code>contexts</code>"]

      1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
         with |navigable id|.

      1. If |navigable| is not a [=/top-level traversable=], return [=error=]
         with [=error code=] [=invalid argument=].

      1. Append |navigable| to |navigables|.

   1. If |navigables| is an empty [=/set=], return [=error=] with [=error code=]
      [=invalid argument=].

1. Let |intercept map| be |session|'s [=intercept map=].

1. Let |parsed patterns| be an empty [=/list=].

1. For each |url pattern| in |url patterns|:

  1. Let |parsed| be the result of [=trying=] to [=parse url pattern=] with |url
     pattern|.

  1. [=list/Append=] |parsed| to |parsed patterns|.

1. Set |intercept map|[|intercept|] to a struct with <code>url patterns</code>
   |parsed patterns|, <code>phases</code> |command
   parameters|["<code>phases</code>"] and <code>browsingContexts</code>
   |navigables|.

1. Return a new [=/map=] matching the
   <code>network.AddInterceptResult</code> production with the
   <code>intercept</code> field set to |intercept|.

</div>

#### The network.continueRequest Command ####  {#command-network-continueRequest}

The <dfn export for=commands>network.continueRequest</dfn> command continues a request
that's blocked by a [=network intercept=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.ContinueRequest = (
        method: "network.continueRequest",
        params: network.ContinueRequestParameters
      )

      network.ContinueRequestParameters = {
        request: network.Request,
        ?body: network.BytesValue,
        ?cookies: [*network.CookieHeader],
        ?headers: [*network.Header],
        ?method: text,
        ?url: text,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.ContinueRequestResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.continueRequest">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |blocked requests| be |session|'s [=blocked request map=].

1. Let |request id| be |command parameters|["<code>request</code>"].

1. If |blocked requests| does not [=map/contain=] |request id| then return
   [=error=] with [=error code=] [=no such request=].

1. Let (|request|, |phase|, <var ignore>response</var>) be |blocked requests|[|request id|].

1. If |phase| is not "<code>beforeRequestSent</code>", then return [=error=]
   with [=error code=] [=invalid argument=].

   Issue: consider a "<code>request already sent</code>" error.

<!-- this has to be first because it's the only fallible operation-->

1. If |command parameters| [=map/contains=] "<code>url</code>":

  1. Let |url record| be the result of applying the [=URL parser=] to |command
     parameters|["<code>url</code>"], with [=base URL=] null.

  1. If |url record| is failure, return [=error=] with [=error code=] [=invalid
     argument=].

     TODO: Should we also resume here?

  1. Let |request|'s [=request/url=] be |url record|.

1. If |command parameters| [=map/contains=] "<code>method</code>":

  1. Let |method| be |command parameters|["<code>method</code>"].

  1. If |method| does not match the [=method token=] production, return [=error=]
     with [=error code=] "<code>invalid argument</code>".

  1. Let |request|'s [=request/method=] be |method|.

1. If |command parameters| [=map/contains=] "<code>headers</code>":

  1. Let |headers| be an empty [=/header list=].

  1. For |header| in |command parameters|["<code>headers</code>"]:

    1. Let |deserialized header| be [=deserialize header=] with |header|.

    1. If |deserialized header|'s name does not match the [=field-name token=]
       production, return [=error=] with [=error code=]
      "<code>invalid argument</code>".

    1. If |deserialized header|'s value does not match the [=header value=]
       production, return [=error=] with [=error code=]
      "<code>invalid argument</code>".

    1. Append |deserialized header| to |headers|.

  1. Set |request|'s [=request/headers list=] to |headers|.

1. If |command parameters| [=map/contains=] "<code>cookies</code>":

  1. Let |cookie header| be an empty [=byte sequence=].

  1. For each |cookie| in |command parameters|["<code>cookies</code>"]:

    1. If |cookie header| is not empty, append `<code>;</code>` to |cookie
       header|.

    1. Append [=serialize cookie header=] with |cookie| to |cookie header|.

  1. Let |found cookie header| be false.

  1. For each |header| in |request|'s [=request/headers list=]:

    1. Let |name| be |header|'s name.

    1. If [=byte-lowercase=] |name| is `<code>cookie</code>`:

      1. Set |header|'s value to |cookie header|.

      1. Set |found cookie header| to true.

      1. Break.

  1. If |found cookie header| is false:

    1. Append the [=header=] (`<code>Cookie</code>`, |cookie header|) to
       |request|'s [=request/headers list=].

1. If |command parameters| [=map/contains=] "<code>body</code>":

  1. Let |body| be [=deserialize protocol bytes=] with |command
     parameters|["<code>body</code>"].

  1. Set |request|'s [=request/body=] to |body|.

1. [=Resume=] with "<code>continue request</code>", |request id|, and (null,
   "<code>incomplete</code>").

1. Return [=success=] with data null.

</div>

#### The network.continueResponse Command ####  {#command-network-continueResponse}

The <dfn export for=commands>network.continueResponse</dfn> command continues a
response that's blocked by a [=network intercept=]. It can be called in the
<code>responseStarted</code> phase, to modify the status and headers of the
response, but still provide the network response body.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.ContinueResponse = (
        method: "network.continueResponse",
        params: network.ContinueResponseParameters
      )

      network.ContinueResponseParameters = {
        request: network.Request,
        ?cookies: [*network.SetCookieHeader]
        ?credentials: network.AuthCredentials,
        ?headers: [*network.Header],
        ?reasonPhrase: text,
        ?statusCode: js-uint,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.ContinueResponseResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.continueResponse">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |request id| be |command parameters|["<code>request</code>"].

1. Let |response| be the result of [=trying=] to [=update the response=] with
   |session|, "<code>continueResponse</code>" and |command parameters|.

1. [=Resume=] with "<code>continue request</code>", |request id|, and
   (|response|, "<code>incomplete</code>").

1. Return [=success=] with data null.

</div>

#### The network.continueWithAuth Command ####  {#command-network-continueWithAuth}

The <dfn export for=commands>network.continueWithAuth</dfn> command continues a
response that's blocked by a [=network intercept=] at the
<code>authRequired</code> phase.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.ContinueWithAuth = (
        method: "network.continueWithAuth",
        params: network.ContinueWithAuthParameters
      )

      network.ContinueWithAuthParameters = {
        request: network.Request,
        (network.ContinueWithAuthCredentials // network.ContinueWithAuthNoCredentials)
      }

      network.ContinueWithAuthCredentials = (
        action: "provideCredentials", <!-- or "provide credentials" or
      "providecredentials" or something else -->
        credentials: network.AuthCredentials
      )

      network.ContinueWithAuthNoCredentials = (
        action: "default" / "cancel"
      )
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.ContinueWithAuthResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.continueWithAuth">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |blocked requests| be |session|'s [=blocked request map=].

1. Let |request id| be |command parameters|["<code>request</code>"].

1. If |blocked requests| does not [=map/contain=] |request id| then return
   [=error=] with [=error code=] [=no such request=].

1. Let (<var ignore>request</var>, |phase|, |response|) be |blocked requests|[|request id|].

1. If |phase| is not "<code>authRequired</code>", then return [=error=]
   with [=error code=] [=invalid argument=].

1. If |command parameters| "<code>action</code>" is "<code>cancel</code>", set
   |response|'s authentication credentials <!--TODO: link--> to
   "<code>cancelled</code>".

1. If |command parameters| "<code>action</code>" is
   "<code>provideCredentials</code>":

   1. Let |credentials| be |command parameters|["<code>credentials</code>"].

   1. Assert: |credentials|["<code>type</code>"] is "<code>password</code>".

   1. Set |response|'s authentication credentials <!--TODO: link--> to
      (|credentials|["<code>username</code>"], |credentials|["<code>password</code>"])

1. [=Resume=] with "<code>continue request</code>", |request id|, and
   (|response|, "<code>incomplete</code>").

1. Return [=success=] with data null.

</div>

#### The network.disownData Command ####  {#command-network-disownData}

The <dfn export for=commands>network.disownData</dfn> command releases a
collected network data for a given [=network/collector=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.DisownData = (
        method: "network.disownData",
        params: network.DisownDataParameters
      )

      network.DisownDataParameters = {
        dataType: network.DataType,
        collector: network.Collector,
        request: network.Request,
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.DisownDataResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.disownData">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |data type| be the value of the "<code>dataType</code>" field in |command
   parameters|.

1. Let |collector id| be the value of the "<code>collector</code>" field in |command
   parameters|.

1. Let |request id| be the value of the "<code>request</code>" field in |command
   parameters|.

1. Let |collectors| be |session|'s [=network collectors=].

1. If |collectors| does not [=map/contain=] |collector id|, return
   [=error=] with [=error code=] [=no such network collector=].

1. Let |collected data| be [=get collected data=] with |request id| and |data type|.

1. If |collected data| is null, return [=error=] with [=error code=] [=no such network data=].

1. [=Remove collector from data=] with |collected data| and |collector id|.

1. Return [=success=] with data [=null=].

</div>

#### The network.failRequest Command ####  {#command-network-failRequest}

The <dfn export for=commands>network.failRequest</dfn> command fails a
fetch that's blocked by a [=network intercept=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.FailRequest = (
        method: "network.failRequest",
        params: network.FailRequestParameters
      )

      network.FailRequestParameters = {
        request: network.Request,
      }
      </pre>
  </dd>
  <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.FailRequestResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.failRequest">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |blocked requests| be |session|'s [=blocked request map=].

1. Let |request id| be |command parameters|["<code>request</code>"].

1. If |blocked requests| does not [=map/contain=] |request id| then return
   [=error=] with [=error code=] [=no such request=].

1. Let (<var ignore>request</var>, |phase|, <var ignore>response</var>) be
   |blocked requests|[|request id|].

1. If |phase| is "<code>authRequired</code>", then return [=error=]
   with [=error code=] [=invalid argument=].

1. Let |response| be a new [=network error=].

    Issue(508): Allow setting the precise kind of error

1. [=Resume=] with "<code>continue request</code>", |request id|, and
   (|response|, "<code>complete</code>").

1. Return [=success=] with data null.

</div>

#### The network.getData Command ####  {#command-network-getData}

The <dfn export for=commands>network.getData</dfn> command retrieves a
network data if it is available.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.GetData = (
        method: "network.getData",
        params: network.GetDataParameters
      )

      network.GetDataParameters = {
        dataType: network.DataType,
        ? collector: network.Collector,
        ? disown: bool .default false,
        request: network.Request,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      network.GetDataResult = {
        bytes: network.BytesValue,
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.getData">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |data type| be |command parameters|["<code>dataType</code>"].

1. Let |request id| be |command parameters|["<code>request</code>"].

1. Let |collector id| be null.

1. If |command parameters| [=map/contains=] "<code>collector</code>":

   1. Let |collectors| be |session|'s [=network collectors=].

   1. If |collectors| does not [=map/contain=] |collector id|, return
      [=error=] with [=error code=] [=no such network collector=].

   1. Set |collector id| to |command parameters|["<code>collector</code>"].

1. Let |disown| be |command parameters|["<code>disown</code>"].

1. If |disown| is true and |collector id| is null, return [=error=]
   with [=error code=] [=invalid argument=].

1. Let |collected data| be [=get collected data=] given |request id| and |data type|.

1. If |collected data| is null:

   1. Return [=error=] with [=error code=] [=no such network data=].

1. If |collected data|'s [=network-data/pending=] is true:

   1. [=Await=] with "network data collected" and (|request id|, |data type|).

1. If |collector id| is not null and if |collected data|'s [=network-data/collectors=]
   does not [=list/contain=] |collector id|:

   1. Return [=error=] with [=error code=] [=no such network data=].

1. Let |bytes| be |collected data|'s [=network-data/bytes=].

1. If |bytes| is null,

   1. Return [=error=] with [=error code=] [=unavailable network data=].

1. Let |body| be a [=/map=] matching the <code>network.GetDataResult</code> production,
   with the <code>bytes</code> field set to |bytes|.

1. If |disown| is true, [=remove collector from data=] with |collected data| and |collector id|.

1. Return [=success=] with data |body|.

</div>


#### The network.provideResponse Command ####  {#command-network-provideResponse}

The <dfn export for=commands>network.provideResponse</dfn> command continues a
request that's blocked by a [=network intercept=], by providing a complete
response.

Note: This will not prevent the request going through the normal request
lifecycle, and therefore emitting other events as it progresses.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      network.ProvideResponse = (
        method: "network.provideResponse",
        params: network.ProvideResponseParameters
      )

      network.ProvideResponseParameters = {
        request: network.Request,
        ?body: network.BytesValue,
        ?cookies: [*network.SetCookieHeader],
        ?headers: [*network.Header],
        ?reasonPhrase: text,
        ?statusCode: js-uint,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.ProvideResponseResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.provideResponse">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |request id| be |command parameters|["<code>request</code>"].

1. Let |response| be the result of [=trying=] to [=update the response=] with
   |session|, "<code>provideResponse</code>", and |command parameters|.

1. If |command parameters| [=map/contains=] "<code>body</code>":

  1. Let |body| be [=deserialize protocol bytes=] with |command
     parameters|["<code>body</code>"].

  1. Set |response|'s [=response/body=] to |body| [=as a body=].

1. [=Resume=] with "<code>continue request</code>", |request id|, and
   (|response|,"<code>complete</code>").

1. Return [=success=] with data null.

</div>

#### The network.removeDataCollector Command ####  {#command-network-removeDataCollector}

The <dfn export for=commands>network.removeDataCollector</dfn> command removes a
[=network/collector=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.RemoveDataCollector = (
        method: "network.removeDataCollector",
        params: network.RemoveDataCollectorParameters
      )

      network.RemoveDataCollectorParameters = {
        collector: network.Collector
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
       network.RemoveDataCollectorResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.removeDataCollector">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |collector id| be the value of the "<code>collector</code>" field in |command
   parameters|.

1. Let |collectors| be |session|'s [=network collectors=].

1. If |collectors| does not [=map/contain=] |collector id|, return
   [=error=] with [=error code=] [=no such network collector=].

1. [=map/Remove=] |collector id| from |session|'s [=network collectors=].

1. For |collected data| in [=collected network data=], [=remove collector from data=]
   with |collected data| and |collector id|.

1. Return [=success=] with data [=null=].

</div>

#### The network.removeIntercept Command ####  {#command-network-removeIntercept}

The <dfn export for=commands>network.removeIntercept</dfn> command removes a
[=network intercept=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.RemoveIntercept = (
        method: "network.removeIntercept",
        params: network.RemoveInterceptParameters
      )

      network.RemoveInterceptParameters = {
        intercept: network.Intercept
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.RemoveInterceptResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for network.removeIntercept">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |intercept| be the value of the "<code>intercept</code>" field in |command
   parameters|.

1. Let |intercept map| be |session|'s [=intercept map=].

1. If |intercept map| does not [=map/contain=] |intercept|, return
   [=error=] with [=error code=] [=no such intercept=].

1. [=map/Remove=] |intercept| from |intercept map|.

Note: removal of an intercept does not affect requests that have been already
blocked by this intercept. Only future requests or future phases of existing
requests will be affected.

1. Return [=success=] with data [=null=].

</div>

#### The network.setCacheBehavior Command ####  {#command-network-setCacheBehavior}

The <dfn export for=commands>network.setCacheBehavior</dfn> command configures
the network cache behavior for certain requests.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.SetCacheBehavior = (
        method: "network.setCacheBehavior",
        params: network.SetCacheBehaviorParameters
      )

      network.SetCacheBehaviorParameters = {
        cacheBehavior: "default" / "bypass",
        ? contexts: [+browsingContext.BrowsingContext]
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.SetCacheBehaviorResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
The <dfn export>WebDriver BiDi cache behavior</dfn> steps given [=/request=] |request| are:

1. Let |navigable| be null.

1. If |request|'s [=request/client=] is an [=environment settings object=]:

  1. Let |environment settings| be |request|'s [=request/client=].

  1. If there is a [=/navigable=] whose [=active window=] is |environment
     settings|' [=environment settings object/global object=], set |navigable|
     to that navigable's [=navigable/top-level traversable=].

1. If |navigable| is not null and [=navigable cache behavior map=] [=set/contains=]
   |navigable|, return [=navigable cache behavior map=][|navigable|].

1. Return [=default cache behavior=].

</div>

<div algorithm>
The <dfn>navigable cache behavior</dfn> steps given |navigable| are:

1. Let |top-level navigable| be |navigable|'s [=navigable/top-level traversable=].

1. If [=navigable cache behavior map=] [=map/contains=] |top-level navigable|, return
   [=navigable cache behavior map=][|top-level navigable|].

1. Return [=default cache behavior=].

</div>

<div algorithm="remote end steps for network.setCacheBehavior">
The [=remote end steps=] given <var ignore>session</var> and |command parameters| are:

1. Let |behavior| be |command parameters|["<code>cacheBehavior</code>"].

1. If |command parameters| does not [=map/contain=] "<code>contexts</code>":

  1. Set the [=default cache behavior=] to |behavior|.

  1. [=map/Clear=] [=navigable cache behavior map=].

  1. Switch on the value of behavior:
    <dl>
      <dt>"<code>bypass</code>"
      <dd>Perform implementation-defined steps to disable any
        implementation-specific resource caches.
      <dt>"<code>default</code>"
      <dd>Perform implementation-defined steps to enable any
        implementation-specific resource caches that are usually enabled in the
        current [=remote end=] configuration.
    </dl>

  1. Return [=success=] with data null.

1. Let |navigables| be an empty [=/set=].

1. For each |navigable id| of |command parameters|["<code>contexts</code>"]:

  1. Let |context| be the result of [=trying=] to [=get a navigable=]
    with |navigable id|.

  1. If |context| is not a [=top-level browsing context=], return [=error=]
     with [=error code=] [=invalid argument=].

  1. [=list/Append=] |context| to |navigables|.

1. For each |navigable| in |navigables|:

    1. If [=navigable cache behavior map=] [=map/contains=] |navigable|, and
       [=navigable cache behavior map=][|navigable|] is equal to |behavior| then
       continue.

    1. Switch on the value of behavior:
      <dl>
        <dt>"<code>bypass</code>"
        <dd>Perform implementation-defined steps to disable any implementation-specific
          resource caches for network requests originating from any browsing
          context for which |navigable| is the [=top-level browsing context=].
        <dt>"<code>default</code>"
        <dd>Perform implementation-defined steps to enable any
          implementation-specific resource caches that are usually enabled in the
          current [=remote end=] configuration for network requests
          originating from any browsing context for which |navigable| is the
          [=top-level browsing context=].
      </dl>

    1. If |behavior| is equal to [=default cache behavior=]:

      1. If [=navigable cache behavior map=] [=map/contains=] |navigable|,
         [=map/remove=] [=navigable cache behavior map=][|navigable|].

    1. Otherwise:

      1. Set [=navigable cache behavior map=][|navigable|] to |behavior|.

1. Return [=success=] with data null.

</div>

#### The network.setExtraHeaders Command ####  {#command-network-setExtraHeaders}

The <dfn export for=commands>network.setExtraHeaders</dfn> command allows
specifying headers that will extend, or overwrite, existing request headers.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      network.SetExtraHeaders = (
        method: "network.setExtraHeaders",
        params: network.SetExtraHeadersParameters
      )

      network.SetExtraHeadersParameters = {
        headers: [*network.Header]
        ? contexts: [+browsingContext.BrowsingContext]
        ? userContexts: [+browser.UserContext]
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.SetExtraHeadersResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm>
To <dfn>update headers</dfn> given |request| and |headers|:

1. Let |request headers| be |request|'s [=request/header list=].

1. For each |header| in |headers|:

  1. [=header list/Set=] |header| in |request headers|.

     Note: This always overwrites the existing value, if any. In particular it
     doesn't append cookies to an existing `Set-Cookie` header.

</div>

<div algorithm>
To <dfn>update request headers</dfn> given |session|, |request|, and |related
navigables|:

1. Assert: |related navigables|'s [=set/size=] is 0 or 1.

   Note: That means this will not work for workers associated with multiple
   navigables. In that case it's unclear in which order to override the headers.

1. [=Update headers=] with |request| and |session|'s
   [=session/extra headers=]' [=extra headers/default headers=]

1. Let |user context headers| be |session|'s [=session/extra headers=]'
   [=extra headers/user context headers=].

1. For |navigable| in |related navigables|:

  1. Let |user context| be |navigable|'s [=associated user context=].

  1. If |user context headers| [=map/contains=] |user context|
     then [=update headers=] with |request| and
     |user context headers|[|user context|]

1. Let |navigable headers| be |session|'s [=session/extra headers=]'
   [=extra headers/navigable headers=].

1. For |navigable| in |related navigables|:

  1. Let |top-level traversable| be |navigable|'s
     [=navigable/top-level traversable=].

  1. If |navigable headers| contains |top-level traversable|
     [=update headers=] with |request| and |navigable headers|[|top-level traversable|].

</div>

<div algorithm="remote end steps for network.setExtraHeaders">
The [=remote end steps=] given |session| and |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |headers| be the result of [=trying=] to [=create a headers list=] with
   |command parameters|["<code>headers</code>"].

1. If |command parameters| [=map/contains=] "<code>userContexts</code>":

  1. Let |user contexts| be an empty [=/list=].

  1. For |user context id| in |command parameters|["<code>userContexts</code>"]:

    1. Let |user context| be [=get user context=] with |user context id|.

    1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

    1. [=list/Append=] |user context| to |user contexts|.

  1. Let |target| be |session|'s [=session/extra headers=]'
     [=extra headers/user context headers=]

  1. For |user context| in |user contexts|:

    1. Set |target|[|user context|] to |headers|.

  1. Return [=success=] with data null.

1. If |command parameters| [=map/contains=] "<code>contexts</code>":

  1. Let |navigables| be the result of [=trying=] to
     [=get valid top-level traversables by ids=]
     with |command parameters|["<code>contexts</code>"].

  1. Let |target| be |session|'s [=session/extra headers=]'
     [=extra headers/navigable headers=]

  1. For |navigable| in |navigables|:

    1. Set |target|[|navigable|] to |headers|.

  1. Return [=success=] with data null.

1. Set |session|'s [=session/extra headers=]'
   [=extra headers/default headers=] to |headers|.

1. Return [=success=] with data null.

</div>

### Events ### {#module-network-event}

#### The network.authRequired Event ####  {#event-network-authRequired}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      network.AuthRequired = (
        method: "network.authRequired",
        params: network.AuthRequiredParameters
      )

      network.AuthRequiredParameters = {
        network.BaseParameters,
        response: network.ResponseData
      }
      </pre>
   </dd>
</dl>

This event is emitted when the user agent is going to prompt for authorization
credentials.

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi auth required</dfn>
steps given [=/request=] |request| and [=/response=] |response|:

1. Let |redirect count| be |request|'s [=redirect count=].

1. Assert: [=before request sent map=][|request|] is equal to |redirect count|.

   Note: This implies that every caller needs to ensure that the [=WebDriver BiDi
   before request sent=] steps are invoked with |request| before these steps.

1. If |request|'s [=request/client=] is not null, let |related navigables|
   be the result of [=get related navigables=] with |request|'s
   [=request/client=]. Otherwise let |related navigables| be an empty
   set.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>network.authRequired</code>" and |related navigables|:

  1. Let |params| be the result of [=process a network event=] with |session|
     "<code>network.authRequired</code>", and |request|.

  1. Let |response data| be the result of [=get the response data=] with |response|.

  1. Assert: |response data| [=map/contains=] "<code>authChallenge</code>".

  1. Set the <code>response</code> field of |params| to |response data|.

  1. Assert: |params| matches the <code>network.AuthRequiredParameters</code>
     production.

  1. Let |body| be a map matching the <code>network.AuthRequired</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

  1. If |params|["<code>isBlocked</code>"] is true:

    1. Let |blocked requests| be |session|'s [=blocked request map=].

    1. Let |request id| be |request|'s [=request id=].

    1. Set |blocked requests|[|request id|] to (|request|,
       "<code>authRequired</code>", |response|).

    <!-- We can ignore the return value here, since we update response -->
    1. [=Await=] with «"<code>continue request</code>"», and |request id|.

    1. [=map/Remove=] |blocked requests|[|request id|].

</div>

#### The network.beforeRequestSent Event #### {#event-network-beforeSendRequest}
<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        network.BeforeRequestSent = (
         method: "network.beforeRequestSent",
         params: network.BeforeRequestSentParameters
        )

       network.BeforeRequestSentParameters = {
         network.BaseParameters,
         ? initiator: network.Initiator,
       }
      </pre>
   </dd>
</dl>

This event is emitted before a request is sent (either over the network or
before it's handled by a serviceworker or a local cache).

<div algorithm>
The steps to check if <dfn>request originates in user context</dfn> given |request|
and |user context| are:

1. Let |settings| be |request|'s [=request/client=].

1. Let |related navigables| be [=get related navigables=] with |settings|.

1. For |navigable| in |related navigables|:

  1. If |navigable|'s [=associated user context=] is |user context| return true.

1. Return false.

</div>

<div algorithm>
The steps to <dfn>get emulated network conditions</dfn> given |related navigables|
are:

1. For each |navigable| of |related navigables|:

   1. Let |top-level traversable| be |navigable|’s [=navigable/top-level traversable=].

   1. Let |user context| be |top-level traversable|'s [=associated user context=].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/navigable network conditions=]
         [=map/contains=] |top-level traversable|, return |session|'s
         [=emulated network conditions=]'s
         [=emulated network conditions/navigable network conditions=][|top-level traversable|].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/user context network conditions=]
         [=map/contains=] |user context|, return |session|'s
         [=emulated network conditions=]'s
         [=emulated network conditions/user context network conditions=][|user context|].

   1. For each |session| in [=active BiDi sessions=]:

      1. If |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/default network conditions=] is not null,
         return |session|'s [=emulated network conditions=]'s
         [=emulated network conditions/default network conditions=].

1. Return null.

</div>

<div algorithm>
The <dfn export>WebDriver BiDi network is offline</dfn> steps given
[=environment settings object=] |settings| are:

1. Let |navigable| be |settings|'s [=relevant global object=]'s
   <a>associated <code>Document</code></a>'s [=/node navigable=].

1. Let |emulated network conditions| be the result of
   [=get emulated network conditions=] with [|navigable|].

1. If |emulated network conditions| is not null and |emulated network conditions|'s
   [=emulated-network-conditions-struct/offline=] is true, return true.

1. Return false.

</div>

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi before request sent</dfn> steps
given [=/request=] |request|:

1. For each |user context| in the [=set of user contexts=]:

   1. If the [=request originates in user context=] steps with |request| and
      |user context| return true:

      1. For each |session| in [=active BiDi sessions=]:

         Note: |user context| can be in not more then one
         [=user context to accept insecure certificates override map=].

         1. If |session|'s [=user context to accept insecure certificates override map=]
            [=map/contains=] |user context|:

            1. Let |accept insecure certificates| be |session|'s
               [=user context to accept insecure certificates override map=][|user context|].

            1. If |accept insecure certificates| is true:

               1. Assert [=endpoint node=] supports accepting insecure TLS
                  connections.

               1. When running the [=Basic Certificate Processing=] steps for
                  |request|, skip step a, along with any other
                  implementation-defined certificate validation steps.

            1. Otherwise, when running the [=Basic Certificate Processing=] steps
               for |request|, perform all steps along with any implementation-defined
               certificate validation steps.

         Note: |user context| can be in not more then one
         [=user context to proxy configuration map=].

         1. If |session|'s [=user context to proxy configuration map=]
            [=set/contains=] |user context|:

            1. Let |proxy configuration| be |session|'s
               [=user context to proxy configuration map=][|user context|].

            1. Take implementation-defined steps to ensure that |request| uses
               the proxy settings defined by the |proxy configuration|.

               Note: the settings are validated when the user context is created and so
               are assumed to be valid at this stage; any error accessing the proxy will
               be reported as a network error when handling the request.

1. [=Maybe collect network request body=] with |request|.

1. If [=before request sent map=] does not contain |request|, set [=before
   request sent map=][|request|] to a new set.

1. Let |redirect count| be |request|'s [=redirect count=].

1. Add |redirect count| to [=before request sent map=][|request|].

1. If |request|'s [=request/client=] is not null, let |related navigables|
   be the result of [=get related navigables=] with |request|'s
   [=request/client=]. Otherwise let |related navigables| be an empty
   set.

1. Let |response| be null.

1. Let |response status| be "<code>incomplete</code>".

1. For each |session| in [=active BiDi sessions=]:

  1. [=Update request headers=] with |session|, |request| and
     |related navigables|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>network.beforeRequestSent</code>" and |related navigables|:

  1. Let |params| be the result of [=process a network event=] with |session|,
     "<code>network.beforeRequestSent</code>", and |request|.

  1. Let |initiator| be the result of [=get the initiator=] with |request|.

  1. If |initiator| is not [=map/is empty|empty=], set the <code>initiator</code> field
     of |params| to |initiator|.

  1. Assert: |params| matches the <code>network.BeforeRequestSentParameters</code>
     production.

  1. Let |body| be a map matching the <code>network.BeforeRequestSent</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

  1. If |params|["<code>isBlocked</code>"] is true, then:

    1. Let |blocked requests| be |session|'s [=blocked request map=].

    1. Let |request id| be |request|'s [=request id=].

    1. Set |blocked requests|[|request id|] to (|request|,
       "<code>beforeRequestSent</code>", null).

    1. Let (|response|, |status|) be [=await=] with «"<code>continue
       request</code>"», and |request|'s [=request id=].

    1. If |status| is "<code>complete</code>" set |response status| to |status|.

    1. [=map/Remove=] |blocked requests|[|request id|].

    Note: While waiting, no further processing of the request occurs.

1. Let |emulated network conditions| be the result of
   [=get emulated network conditions=] with |related navigables|.

1. If |emulated network conditions| is not null and
   |emulated network conditions|'s [=emulated-network-conditions-struct/offline=] is
   true, return ([=network error=], "<code>complete</code>").

1. Return (|response|, |response status|).

Issue: Respect return value in Fetch's "HTTP-network-or-cache fetch" algorithm.

</div>

#### The network.fetchError Event #### {#event-network-fetchError}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        network.FetchError = (
         method: "network.fetchError",
         params: network.FetchErrorParameters
        )

       network.FetchErrorParameters = {
         network.BaseParameters,
         errorText: text,
       }
      </pre>
   </dd>
</dl>

This event is emitted when a network request ends in an error.

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi fetch
error</dfn> steps given [=/request=] |request|:

1. If [=before request sent map=][|request|] does not contain |request|'s
   [=redirect count=], then run the [=WebDriver BiDi before request sent=] steps
   with |request|.

   Note: This ensures that a <code>network.beforeRequestSent</code> can
   always be emitted before a <code>network.fetchError</code>, without the
   caller needing to explicitly invoke the [=WebDriver BiDi before request
   sent=] steps on every error path.

1. If |request|'s [=request/client=] is not null, let |related navigables|
   be the result of [=get related navigables=] with |request|'s
   [=request/client=]. Otherwise let |related navigables| be an empty
   set.

1. [=Maybe abort network response body collection=] with |request|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>network.fetchError</code>" and |related navigables|:

  1. Let |params| be the result of [=process a network event=] with |session|
     "<code>network.fetchError</code>", and |request|.

  1. Set the <code>errorText</code> field of |params| to an implementation-defined string
     describing the error which caused the request to be aborted.

  1. Assert: |params| matches the <code>network.FetchErrorParameters</code>
     production.

  1. Let |body| be a map matching the <code>network.FetchError</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The network.responseCompleted Event #### {#event-network-responseCompleted}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        network.ResponseCompleted = (
         method: "network.responseCompleted",
         params: network.ResponseCompletedParameters
        )

       network.ResponseCompletedParameters = {
         network.BaseParameters,
         response: network.ResponseData,
       }
      </pre>
   </dd>
</dl>

This event is emitted after the full response body is received.

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi response completed</dfn> steps
given [=/request=] |request| and [=/response=] |response|:

1. Let |redirect count| be |request|'s [=redirect count=].

1. Assert: [=before request sent map=][|request|] contains |redirect count|.

   Note: This implies that every caller needs to ensure that the [=WebDriver BiDi
   before request sent=] steps are invoked with |request| before these steps.

1. If |request|'s [=request/client=] is not null, let |related navigables|
   be the result of [=get related navigables=] with |request|'s
   [=request/client=]. Otherwise let |related navigables| be an empty
   set.

1. [=Maybe collect network response body=] with |request| and |response|.

1. Let |sessions| be the [=set of sessions for which an event is enabled=]
   given "<code>network.responseCompleted</code>" and |related navigables|.

1. For each |session| in |sessions|:

  1. Let |params| be the result of [=process a network event=] with |session|
     "<code>network.responseCompleted</code>", and |request|.

  1. Assert: |params|["<code>isBlocked</code>"] is false.

  1. Let |response data| be the result of [=get the response data=] with |response|.

  1. Set the <code>response</code> field of |params| to |response data|.

  1. Assert: |params| matches the <code>network.ResponseCompletedParameters</code>
     production.

  1. Let |body| be a map matching the <code>network.ResponseCompleted</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The network.responseStarted Event #### {#event-network-responseStarted}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        network.ResponseStarted = (
         method: "network.responseStarted",
         params: network.ResponseStartedParameters
        )

       network.ResponseStartedParameters = {
         network.BaseParameters,
         response: network.ResponseData,
       }
      </pre>
   </dd>
</dl>

This event is emitted after the response headers are received but before the
body is complete.

<div algorithm>
The [=remote end event trigger=] is the <dfn export>WebDriver BiDi response started</dfn> steps
given [=/request=] |request| and [=/response=] |response|:

1. Let |redirect count| be |request|'s [=redirect count=].

1. Assert: [=before request sent map=][|request|] is equal to |redirect count|.

   Note: This implies that every caller needs to ensure that the [=WebDriver BiDi
   before request sent=] steps are invoked with |request| before these steps.

1. If |request|'s [=request/client=] is not null, let |related navigables|
   be the result of [=get related navigables=] with |request|'s
   [=request/client=]. Otherwise let |related navigables| be an empty
   set.

1. Let |response status| be "<code>incomplete</code>".

1. Let |sessions| be the [=set of sessions for which an event is enabled=]
   given "<code>network.responseStarted</code>" and |related navigables|.

1. For each |session| in |sessions|:

  1. Let |params| be the result of [=process a network event=] with |session|
     "<code>network.responseStarted</code>", and |request|.

  1. Let |response data| be the result of [=get the response data=] with |response|.

  1. Set the <code>response</code> field of |params| to |response data|.

  1. Assert: |params| matches the <code>network.ResponseStartedParameters</code>
     production.

  1. Let |body| be a map matching the <code>network.ResponseStarted</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

  1. If |params|["<code>isBlocked</code>"] is true:

    1. Let |blocked requests| be |session|'s [=blocked request map=].

    1. Let |request id| be |request|'s [=request id=].

    1. Set |blocked requests|[|request id|] to (|request|,
       "<code>beforeRequestSent</code>", |response|).

    1. Let (|response|, |status|) be [=await=] with «"<code>continue request</code>"», and
       |request id|.

    1. If |status| is "<code>complete</code>", set |response status| to |status|.

    1. [=map/Remove=] |blocked requests|[|request id|].

1. Return (|response|, |response status|).

</div>

## The script Module ## {#module-script}

The <dfn export for=modules>script</dfn> module contains commands and events
relating to script realms and execution.

### Definition ### {#module-script-definition}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
ScriptCommand = (
  script.AddPreloadScript //
  script.CallFunction //
  script.Disown //
  script.Evaluate //
  script.GetRealms //
  script.RemovePreloadScript
)
</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
ScriptResult = (
  script.AddPreloadScriptResult /
  script.CallFunctionResult /
  script.DisownResult /
  script.EvaluateResult /
  script.GetRealmsResult /
  script.RemovePreloadScriptResult
)

ScriptEvent = (
  script.Message //
  script.RealmCreated //
  script.RealmDestroyed
)
</pre>

### Preload Scripts ### {#preload-scripts}

A <dfn>Preload script</dfn> is one which runs on creation of a new {{Window}},
before any author-defined script have run.

TODO: Extend this to scripts in other kinds of realms.

A [=BiDi session=] has a <dfn>preload script map</dfn> which is a [=/map=] in
which the keys are [[!RFC9562|UUID]]s, and the values are [=structs=] with
an [=struct/item=] named <code>function declaration</code>, which is a string,
an [=struct/item=] named <code>arguments</code>, which is a list,
an [=struct/item=] named <code>contexts</code>, which is a list or null,
an [=struct/item=] named <code>sandbox</code>, which is a string or null,
and an [=struct/item=] named <code>user contexts</code>, which is a [=/set=].

Note: If executing a [=preload script=] fails, either due to a syntax error, or
a runtime exception, an [[ECMAScript]] exception is reported in the realm in
which it was being executed, and other preload scripts run as normal.

<div algorithm>
To <dfn export>run WebDriver BiDi preload scripts</dfn> given |environment settings|:

1. Let |document| be |environment settings|' [=relevant global object=]'s
   <a>associated <code>Document</code></a>.

1. Let |navigable| be |document|'s [=/navigable=].

1. Let |user context| be |navigable|'s [=associated user context=].

1. Let |user context id| be |user context|'s [=user context id=].

1. For each |session| in [=active BiDi sessions=]:

  1. For each |preload script| in |session|'s [=preload script map=]'s
     [=map/values=]:

    1. If |preload script|'s <code>user contexts</code>'s [=set/size=] is not zero:

       1. If |preload script|'s <code>user contexts</code> does not [=list/contain=] |user context id|,
          [=continue=].

    1. If |preload script|'s <code>contexts</code> is not null:

       1. Let |navigable id| be |navigable|’s [=navigable/top-level traversable=]'s id.

       1. If |preload script|'s <code>contexts</code> does not [=list/contain=] |navigable id|,
          [=continue=].

    1. If |preload script|'s <code>sandbox</code> is not null, let |realm| be [=get
       or create a sandbox realm=] with |preload script|'s <code>sandbox</code> and
       |navigable|. Otherwise let |realm| be |environment settings|'
       [=realm execution context=]'s Realm component.

    1. Let |exception reporting global| be be |environment settings|'
       [=realm execution context=]'s Realm component's [=realm/global object=].

    1. Let |arguments| be |preload script|'s <code>arguments</code>.

    1. Let |deserialized arguments| be an empty list.

    1. For each |argument| in |arguments|:

      1. Let |channel| be [=create a channel=] with |session|, |realm| and
         |argument|.

      1. Append |channel| to |deserialized arguments|.

    1. Let |base URL| be the [=API base URL=] of |environment settings|.

    1. Let |options| be the [=default script fetch options=].

    1. Let |function declaration| be |preload script|'s <code>function declaration</code>.

    1. Let |function body evaluation status| be the result of
       [=evaluate function body=] with |function declaration|,
       |environment settings|, |base URL|, and |options|.

    1. If |function body evaluation status| is an [=abrupt completion=], then
       [=report an exception=] given by |function body evaluation status|.\[[Value]]
       for |exception reporting global|.

    1. Let |function object| be |function body evaluation status|.\[[Value]].

    1. If [=IsCallable=](|function object|) is <code>false</code>:

       1. Let |error| be a new <a data-link-type="exception">TypeError</a> object in |realm|.

       1. [=Report an exception=] |error| for |exception reporting global|.

    1. [=Prepare to run script=] with |environment settings|.

    1. Set |evaluation status| to
       [=Call=](|function object|, null, |deserialized arguments|).

    1. [=Clean up after running script=] with |environment settings|.

    1. If |evaluation status| is an [=abrupt completion=], then
       [=report an exception=] given by |evaluation status|.\[[Value]] for
       |exception reporting global|.

</div>

### Types ### {#module-script-types}

#### The script.Channel Type #### {#type-script-Channel}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.Channel = text;
</pre>

The <code><dfn>script.Channel</dfn></code> type represents the id of a specific channel
used to send custom messages from the [=remote end=] to the [=local end=].

#### The script.ChannelValue Type #### {#type-script-ChannelValue}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.ChannelValue = {
  type: "channel",
  value: script.ChannelProperties,
}

script.ChannelProperties = {
  channel: script.Channel,
  ? serializationOptions: script.SerializationOptions,
  ? ownership: script.ResultOwnership,
}
</pre>

The <code>script.ChannelValue</code> type represents an
<code>ArgumentValue</code> that can be deserialized into a function that sends
messages from the [=remote end=] to the [=local end=].

<div algorithm>

To <dfn>create a channel</dfn> given |session|, |realm| and |protocol value|:

1. Let |channel properties| be |protocol value|["<code>value</code>"].

1. Let |steps| be the following steps given the argument |message|:

  1. Let |current realm| be the [=current Realm Record=].

  1. [=Emit a script message=] with |session|, |current realm|, |channel properties| and |message|.

1. Return [=CreateBuiltinFunction=](|steps|, 1, "", « », |realm|).

</div>

#### The script.EvaluateResult Type #### {#type-script-EvaluateResult}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.EvaluateResult = (
  script.EvaluateResultSuccess /
  script.EvaluateResultException
)

script.EvaluateResultSuccess = {
  type: "success",
  result: script.RemoteValue,
  realm: script.Realm
}

script.EvaluateResultException = {
  type: "exception",
  exceptionDetails: script.ExceptionDetails
  realm: script.Realm
}
</pre>

The <code>script.EvaluateResult</code> type indicates the return value of a command
that executes script. The <code>script.EvaluateResultSuccess</code> variant is used in
cases where the script completes normally and the
<code>script.EvaluateResultException</code> variant is used in cases where the script
completes with a thrown exception.

#### The script.ExceptionDetails Type #### {#type-script-ExceptionDetails}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.ExceptionDetails = {
  columnNumber: js-uint,
  exception: script.RemoteValue,
  lineNumber: js-uint,
  stackTrace: script.StackTrace,
  text: text,
}
</pre>

The <code>script.ExceptionDetails</code> type represents a JavaScript exception.

<div algorithm>

To <dfn>get exception details</dfn> given a |realm|, a [=completion record=]
|record|, an |ownership type| and a |session|:

1. Assert: |record|.\[[Type]] is <code>throw</code>.

1. Let |text| be an implementation-defined textual description of the error
   represented by |record|.

   TODO: Tighten up the requirements here; people will probably try to parse
   this data with regex or something equally bad.

1. Let |serialization options| be a [=/map=] matching the
   <code>script.SerializationOptions</code> production with the fields set to
   their default values.

1. Let |exception| be the result of [=serialize as a remote value=] with
   |record|.\[[Value]], |serialization options|, |ownership type|,
   a new [=/map=] as serialization internal map, |realm| and |session|.

1. Let |stack trace| be the [=stack trace for an exception=] given |record|.

1. If |stack trace| has size of 1 or greater, let |line number| be value of the
   <code>lineNumber</code> field in |stack trace|[0], and let |column number| be
   the value of the <code>columnNumber</code> field |stack trace|[0]. Otherwise
   let |line number| and |column number| be 0.

1. Let |exception details| be a [=/map=] matching the
   <code>script.ExceptionDetails</code> production, with the <code>text</code>
   field set to |text|, the <code>exception</code> field set to |exception|, the
   <code>lineNumber</code> field set to |line number|, the
   <code>columnNumber</code> field set to |column number|, and the
   <code>stackTrace</code> field set to |stack trace|.

1. Return |exception details|.

</div>

#### The script.Handle Type #### {#type-script-Handle}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.Handle = text;
</pre>

The <code>script.Handle</code> type represents a handle to an object owned by
the ECMAScript runtime. The handle is only valid in a specific [=Realm=].

Each ECMAScript [=Realm=] has a corresponding <dfn>handle object map</dfn>. This is a
strong [=/map=] from handle ids to their corresponding objects.

#### The script.InternalId Type #### {#type-script-InternalId}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.InternalId = text;
</pre>

The <code>script.InternalId</code> type represents the id of
a previously serialized <code>script.RemoteValue</code> during
[=serialize as a remote value|serialization=].


#### The script.LocalValue Type #### {#type-script-LocalValue}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.LocalValue = (
  script.RemoteReference /
  script.PrimitiveProtocolValue /
  script.ChannelValue /
  script.ArrayLocalValue /
  { script.DateLocalValue } /
  script.MapLocalValue /
  script.ObjectLocalValue /
  { script.RegExpLocalValue } /
  script.SetLocalValue
)

script.ListLocalValue = [*script.LocalValue];

script.ArrayLocalValue = {
  type: "array",
  value: script.ListLocalValue,
}

script.DateLocalValue = (
  type: "date",
  value: text
)

script.MappingLocalValue = [*[(script.LocalValue / text), script.LocalValue]];

script.MapLocalValue = {
  type: "map",
  value: script.MappingLocalValue,
}

script.ObjectLocalValue = {
  type: "object",
  value: script.MappingLocalValue,
}

script.RegExpValue = {
  pattern: text,
  ? flags: text,
}

script.RegExpLocalValue = (
  type: "regexp",
  value: script.RegExpValue,
)

script.SetLocalValue = {
  type: "set",
  value: script.ListLocalValue,
}
</pre>

The <code><dfn export>script.LocalValue</dfn></code> type represents values which can be
deserialized into ECMAScript. This includes both primitive and non-primitive
values as well as [=script.RemoteReference|remote references=] and
[=script.Channel|channels=].

<div algorithm>

To <dfn>deserialize key-value list</dfn> given |serialized key-value list|, |realm|
and |session|:

1. Let |deserialized key-value list| be a new list.

1. For each |serialized key-value| in the |serialized key-value list|:

   1. If [=list/size=] of |serialized key-value| is not 2,
      return [=error=] with [=error code=] [=invalid argument=].

   1. Let |serialized key| be |serialized key-value|[0].

   1. If |serialized key| is a <code>string</code>,
      let |deserialized key| be |serialized key|.

   1. Otherwise let |deserialized key| be result of [=trying=] to
      given [=deserialize local value=] with |serialized key|, |realm| and |session|.

   1. Let |serialized value| be |serialized key-value|[1].

   1. Let |deserialized value| be result of [=trying=] to [=deserialize local value=]
      given |serialized value|, |realm| and |session|.

   1. Append [=CreateArrayFromList=](«|deserialized key|, |deserialized value|») to
      |deserialized key-value list|.

1. Return [=success=] with data |deserialized key-value list|.

</div>
<div algorithm>

To <dfn>deserialize value list</dfn> given |serialized value list|, |realm| and
|session|:

1. Let |deserialized values| be a new list.

1. For each |serialized value| in the |serialized value list|:

   1. Let |deserialized value| be result of [=trying=] to [=deserialize local value=]
      given |serialized value|, |realm| and |session|.

   1. Append |deserialized value| to |deserialized values|;

1. Return [=success=] with data |deserialized values|.

</div>

<div algorithm>

To <dfn>deserialize local value</dfn> given |local protocol value|, |realm| and
|session|:

1. If |local protocol value| matches the [=script.RemoteReference=] production, return
   [=deserialize remote reference=] of given |local protocol value|, |realm| and
   |session|.

1. If |local protocol value| matches the [=script.PrimitiveProtocolValue=] production, return
   [=deserialize primitive protocol value=] with |local protocol value|.

1. If |local protocol value| matches the <code>script.ChannelValue</code> production,
   return [=create a channel=] with |session|, |realm| and |local protocol value|.

1. Let |type| be the value of the <code>type</code> field of |local protocol value| or undefined if
   no such a field.

1. Let |value| be the value of the <code>value</code> field of |local protocol value| or undefined
   if no such a field.

1. In the following list of conditions and associated steps, run the first set of steps for which
   the associated condition is true:

  <dl>

    <dt>|type| is the string "<code>array</code>"
    <dd>
      1. Let |deserialized value list| be a result of [=trying=] to [=deserialize value list=]
         given |value|, |realm| and |session|.

      1. Return [=success=] with data [=CreateArrayFromList=](|deserialized value list|).

    <dt>|type| is the string "<code>date</code>"
    <dd>
      1. If |value| does not match [=Date Time String Format=], return [=error=] with
         [=error code=] [=invalid argument=].

      1. Let |date result| be [=Construct=]([=Date=], |value|).

      1. Assert: |date result| is not an [=abrupt completion=].

      1. Return [=success=] with data |date result|.

    <dt>|type| is the string "<code>map</code>"
    <dd>
      1. Let |deserialized key-value list| be a result of [=trying=] to
         [=deserialize key-value list=] with |value|, |realm| and |session|.

      1. Let |iterable| be [=CreateArrayFromList=](|deserialized key-value list|)

      1. Return [=success=] with data [=constructor/Map=](|iterable|).

    <dt>|type| is the string "<code>object</code>"
    <dd>
      1. Let |deserialized key-value list| be a result of [=trying=] to
         [=deserialize key-value list=] with |value|, |realm| and |session|.

      1. Let |iterable| be [=CreateArrayFromList=](|deserialized key-value list|)

      1. Return [=success=] with data [=Object.fromEntries=](|iterable|).

    <dt>|type| is the string "<code>regexp</code>"
    <dd>
      1. Let |pattern| be the value of the <code>pattern</code> field of |local protocol value|.

      1. Let |flags| be the value of the <code>flags</code> field of |local protocol value| or
         undefined if no such a field.

      1. Let |regex_result| be [=Regexp=](|pattern|, |flags|). If this throws exception, return [=error=]
         with [=error code=] [=invalid argument=].

      1. Return [=success=] with data |regex_result|.

    <dt>|type| is the string "<code>set</code>"
    <dd>
      1. Let |deserialized value list| be a result of [=trying=] to [=deserialize value list=]
         given |value|, |realm| and |session|.

      1. Let |iterable| be [=CreateArrayFromList=](|deserialized key-value list|)

      1. Return [=success=] with data [=Set object=](|iterable|).

    <dt>otherwise
    <dd>Return [=error=] with [=error code=] [=invalid argument=].
  </dl>

</div>


#### The script.PreloadScript Type #### {#type-script-PreloadScript}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.PreloadScript = text;
</pre>

The <code>script.PreloadScript</code> type represents a handle to a script that will run
on realm creation.

#### The script.Realm Type #### {#type-script-Realm}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.Realm = text;
</pre>

Each [=realm=] has an associated <dfn export>realm id</dfn>, which is a string
uniquely identifying that realm. This is implicitly set when the realm is
created.

The [=realm id=] for a realm is opaque and must not be derivable from the handle
id of the corresponding global object in the [=handle object map=] or, where
relevant, from the [=navigable id=] of any [=/navigable=].

Note: this is to ensure that users do not rely on implementation-specific
relationships between different ids.

<div algorithm>
To <dfn>get a realm</dfn> given |realm id|:

1. If |realm id| is null, return [=success=] with data null.

1. If there is no [=realm=] with [=realm id|id=] |realm id| return
   [=error=] with [=error code=] [=no such frame=]

1. Let |realm| be the [=realm=] with [=realm id|id=] |realm id|.

1. Return [=success=] with data |realm|

Issue: This has the wrong error code
</div>

#### The script.PrimitiveProtocolValue Type #### {#type-script-PrimitiveProtocolValue}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.PrimitiveProtocolValue = (
  script.UndefinedValue /
  script.NullValue /
  script.StringValue /
  script.NumberValue /
  script.BooleanValue /
  script.BigIntValue
)

script.UndefinedValue = {
  type: "undefined",
}

script.NullValue = {
  type: "null",
}

script.StringValue = {
  type: "string",
  value: text,
}

script.SpecialNumber = "NaN" / "-0" / "Infinity" / "-Infinity";

script.NumberValue = {
  type: "number",
  value: number / script.SpecialNumber,
}

script.BooleanValue = {
  type: "boolean",
  value: bool,
}

script.BigIntValue = {
  type: "bigint",
  value: text,
}
</pre>

The <dfn>script.PrimitiveProtocolValue</dfn> represents values which can only be
represented by value, never by reference.

<div algorithm>

To <dfn>serialize primitive protocol value</dfn> given a |value|:

1. Let |remote value| be undefined.

1. In the following list of conditions and associated steps, run the first set
   of steps for which the associated condition is true, if any:

  <dl>
    <dt>[=/Type=](|value|) is undefined
    <dd>Let |remote value| be a [=/map=] matching the <code>script.UndefinedValue</code>
    production in the {^local end definition^}.

    <dt>[=/Type=](|value|) is Null
    <dd>Let |remote value| be a [=/map=] matching the <code>script.NullValue</code>
    production in the {^local end definition^}.

    <dt>[=/Type=](|value|) is String
    <dd>Let |remote value| be a [=/map=] matching the <code>script.StringValue</code>
    production in the {^local end definition^}, with the <code>value</code>
    property set to |value|.

    Issue: This doesn't handle lone surrogates

    <dt>[=/Type=](|value|) is Number
    <dd>
    1. Switch on the value of |value|:
      <dl>
        <dt>NaN
        <dd>Let |serialized| be <code>"NaN"</code>
        <dt>-0
        <dd>Let |serialized| be <code>"-0"</code>
        <dt>Infinity
        <dd>Let |serialized| be <code>"Infinity"</code>
        <dt>-Infinity
        <dd>Let |serialized| be <code>"-Infinity"</code>
        <dt>Otherwise:
        <dd>Let |serialized| be |value|
      </dl>

    1. Let |remote value| be a [=/map=] matching the <code>script.NumberValue</code>
       production in the {^local end definition^}, with the <code>value</code>
       property set to |serialized|.

    <dt>[=/Type=](|value|) is Boolean
    <dd>Let |remote value| be a [=/map=] matching the <code>script.BooleanValue</code>
        production in the {^local end definition^}, with the <code>value</code>
        property set to |value|.

    <dt>[=/Type=](|value|) is BigInt
    <dd>Let |remote value| be a [=/map=] matching the <code>script.BigIntValue</code>
        production in the {^local end definition^}, with the <code>value</code>
        property set to the result of running the [=ToString=] operation on
        |value|.

  </dl>

1. Return |remote value|

</div>


<div algorithm>

To <dfn>deserialize primitive protocol value</dfn> given a |primitive protocol value|:

1. Let |type| be the value of the <code>type</code> field of |primitive protocol value|.

1. Let |value| be undefined.

1. If |primitive protocol value| has field <code>value</code>:
   1. Let |value| be the value of the <code>value</code> field of |primitive protocol value|.

1. In the following list of conditions and associated steps, run the first set of steps for which
   the associated condition is true:

  <dl>

    <dt>|type| is the string "<code>undefined</code>"
    <dd>Return [=success=] with data [=undefined=].

    <dt>|type| is the string "<code>null</code>"
    <dd>Return [=success=] with data [=null=].

    <dt>|type| is the string "<code>string</code>"
    <dd>Return [=success=] with data |value|.

    <dt>|type| is the string "<code>number</code>"
    <dd>
      1. If [=/Type=](|value|) is Number, return [=success=] with data |value|.

      1. Assert: [=/Type=](|value|) is String.

      1. If |value| is the string "<code>NaN</code>", return [=success=] with
         data NaN.

      1. Let |number_result| be [=StringToNumber=](|value|).

      1. If |number_result| is NaN, return [=error=] with [=error code=] [=invalid argument=]

      1. Return [=success=] with data |number_result|.

    <dt>|type| is the string "<code>boolean</code>"
    <dd>Return [=success=] with data |value|.

    <dt>|type| is the string "<code>bigint</code>"
    <dd>
      1. Let |bigint_result| be [=StringToBigInt=](|value|).

      1. If |bigint_result| is undefined, return [=error=] with [=error code=] [=invalid argument=]

      1. Return [=success=] with data |bigint_result|.

  </dl>

1. Return [=error=] with [=error code=] [=invalid argument=]

</div>

#### The script.RealmInfo Type ####  {#type-script-RealmInfo}

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
script.RealmInfo = (
  script.WindowRealmInfo /
  script.DedicatedWorkerRealmInfo /
  script.SharedWorkerRealmInfo /
  script.ServiceWorkerRealmInfo /
  script.WorkerRealmInfo /
  script.PaintWorkletRealmInfo /
  script.AudioWorkletRealmInfo /
  script.WorkletRealmInfo
)

script.BaseRealmInfo = (
  realm: script.Realm,
  origin: text
)

script.WindowRealmInfo = {
  script.BaseRealmInfo,
  type: "window",
  context: browsingContext.BrowsingContext,
  ? userContext: browser.UserContext,
  ? sandbox: text
}

script.DedicatedWorkerRealmInfo = {
  script.BaseRealmInfo,
  type: "dedicated-worker",
  owners: [script.Realm]
}

script.SharedWorkerRealmInfo = {
  script.BaseRealmInfo,
  type: "shared-worker"
}

script.ServiceWorkerRealmInfo = {
  script.BaseRealmInfo,
  type: "service-worker"
}

script.WorkerRealmInfo = {
  script.BaseRealmInfo,
  type: "worker"
}

script.PaintWorkletRealmInfo = {
  script.BaseRealmInfo,
  type: "paint-worklet"
}

script.AudioWorkletRealmInfo = {
  script.BaseRealmInfo,
  type: "audio-worklet"
}

script.WorkletRealmInfo = {
  script.BaseRealmInfo,
  type: "worklet"
}
</pre>

Note: there's a 1:1 relationship between the <code>script.RealmInfo</code>
      variants and values of <code>script.RealmType</code>.

The <code>script.RealmInfo</code> type represents the properties of a realm.

<div algorithm>
To <dfn>get the navigable</dfn> with given |realm|:

1. Let |global object| be the [=realm/global object=] of the |realm|.

1. Let |global object| be the [=unwrapped=] |global object|.

1. If |global object| is not a {{Window}} object, return <code>null</code>.

1. Let |document| be |global object|'s wrapped {{Window}}'s
   <a>associated <code>Document</code></a>.

1. Return |document|'s [=/node navigable=].

</div>
<div algorithm>
To <dfn>get the worker's owners</dfn> with given |global object|:

1. Assert: |global object| is a {{WorkerGlobalScope}} object.

1. Let |owners| be an empty [=/list=].

1. For each |owner| in the |global object|'s associated [=owner set=]:

   1. Let |owner environment settings| be |owner|'s [=relevant settings object=].

   1. Let |owner realm info| be the result of [=get the realm info=] given
      |owner environment settings|.

   1. If |owner realm info| is null, continue.

   1. Append |owner realm info|["<code>id</code>"] to |owners|.

1. Return |owners|.

</div>

<div algorithm>
To <dfn>get the realm info</dfn> given |environment settings|:

1. Let |realm| be |environment settings|' [=realm execution context=]'s Realm component.

1. Let |realm id| be the [=realm id=] for |realm|.

1. Let |origin| be the [=serialization of an origin=] given |environment settings|'s |origin|.

1. Let |global object| be the [=realm/global object=] specified by |environment settings|

1. Run the steps under the first matching condition:

  <dl>
    <dt>|global object| is a {{Window}} object
    <dd>
      1. Let |document| be |environment settings|' [=relevant global object=]'s
         <a>associated <code>Document</code></a>.
      1. Let |navigable| be |document|'s [=/node navigable=].
      1. If |navigable| is null, return null.
      1. Let |navigable id| be the [=navigable id=] for |navigable|.
      1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=]
      1. Let |realm info| be a [=/map=] matching the <code>script.WindowRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, the <code>origin</code>
         field set to |origin|, the <code>context</code> field set to |navigable id| and the
         <code>userContext</code> field set to |user context id|.

    <dt>|global object| is {{SandboxWindowProxy}} object
    <dd>
      TODO: Unclear if this is the right formulation for handling sandboxes.

      1. Let |document| be |global object|'s wrapped {{Window}}'s
         <a>associated <code>Document</code></a>.
      1. Let |navigable| be |document|'s [=/node navigable=].
      1. If |navigable| is null, return null.
      1. Let |navigable id| be the [=navigable id=] for |navigable|.
      1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=]
      1. Let |sandbox name| be the result of [=get a sandbox name=] given
         |realm|.
      1. Assert: |sandbox name| is not null.
      1. Let |realm info| be a [=/map=] matching the <code>script.WindowRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, the <code>origin</code>
         field set to |origin|, the <code>context</code> field set to |navigable id|,
         the <code>userContext</code> field set to |user context id|, and the
         <code>sandbox</code> field set to |sandbox name|.


    <dt>|global object| is a {{DedicatedWorkerGlobalScope}} object
    <dd>
      1. Let |owners| be the result of [=get the worker's owners=] given |global object|.
      1. Assert: |owners| has precisely one item.
      1. Let |realm info| be a [=/map=] matching the <code>script.DedicatedWorkerRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, the <code>origin</code> field
         set to |origin|, and the <code>owners</code> field set to |owners|.

    <dt>|global object| is a {{SharedWorkerGlobalScope}} object
    <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.SharedWorkerRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code> field
         set to |origin|.

    <dt>|global object| is a {{ServiceWorkerGlobalScope}} object
    <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.ServiceWorkerRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code> field
         set to |origin|.

   <dt>|global object| is a {{WorkerGlobalScope}} object
   <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.WorkerRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code>
         field set to |origin|.

   <dt>|global object| is a {{PaintWorkletGlobalScope}} object
   <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.PaintWorkletRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code>
         field set to |origin|.

   <dt>|global object| is a {{AudioWorkletGlobalScope}} object
   <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.AudioWorkletRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code>
         field set to |origin|.

   <dt>|global object| is a {{WorkletGlobalScope}} object
   <dd>
      1. Let |realm info| be a [=/map=] matching the <code>script.WorkletRealmInfo</code> production,
         with the <code>realm</code> field set to |realm id|, and the <code>origin</code>
         field set to |origin|.

   <dt>Otherwise:
   <dd>
      1. Let |realm info| be null.
  </dl>

1. Return |realm info|

Note: Future variations of this specification will retain the invariant that
         the last component of the type name after splitting on "<code>-</code>"
         will always be "<code>worker</code>" for globals implementing
         {{WorkerGlobalScope}}, and "<code>worklet</code>" for globals
         implementing {{WorkletGlobalScope}}.
</div>

#### The script.RealmType Type ####  {#type-script-RealmType}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.RealmType = "window" / "dedicated-worker" / "shared-worker" / "service-worker" /
                   "worker" / "paint-worklet" / "audio-worklet" / "worklet"
</pre>

The <code>script.RealmType</code> type represents the different types of Realm.

#### The script.RemoteReference Type #### {#type-script-RemoteReference}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
<!-- This is specifically ordered in the order in which matches need to be -->
<!-- evaluated, since the definitions are overlapping -->
script.RemoteReference = (
  script.SharedReference /
  script.RemoteObjectReference
)

script.SharedReference = {
   sharedId: script.SharedId
   <!-- Ensure that if we have a handle, it at least has the correct type -->
   ? handle: script.Handle,
   Extensible
}

script.RemoteObjectReference = {
   handle: script.Handle,
   <!-- This shouldn't ever match. The problem is that Extensible would
   otherwise allow this to match a non-text sharedId -->
   ? sharedId: script.SharedId
   Extensible
}
</pre>

The <code><dfn>script.RemoteReference</dfn></code> type is either a
<code>script.RemoteObjectReference</code> representing a remote reference to an
existing ECMAScript object in [=handle object map=] in the given [=Realm=], or
is a <code>script.SharedReference</code> representing a reference to a [=node=].

Issue: handle "stale object reference" case.

Note: if the provided reference has both <code>handle</code> and
<code>sharedId</code>, the algorithm will ignore <code>handle</code> and respect
only <code>sharedId</code>.

<div algorithm>
To <dfn>deserialize remote reference</dfn> given |remote reference|, |realm| and
|session|:

1. Assert |remote reference| matches the <code>script.RemoteReference</code> production.

1. If |remote reference| matches the <code>script.SharedReference</code> production, return
   [=deserialize shared reference=] with |remote reference|, |realm| and |session|.

1. Return [=deserialize remote object reference=] with |remote reference| and
   |realm|.

</div>

<div algorithm>
To <dfn>deserialize remote object reference</dfn> given |remote object reference| and
|realm|:

1. Let |handle id| be the value of the <code>handle</code> field of
   |remote object reference|.

1. Let |handle map| be |realm|'s [=handle object map=]

1. If |handle map| does not contain |handle id|, then return [=error=] with
   [=error code=] [=no such handle=].

1. Return [=success=] with data |handle map|[|handle id|].

</div>

<div algorithm>
To <dfn>deserialize shared reference</dfn> given |shared reference|, |realm| and
|session|:

1. Assert |shared reference| matches the <code>script.SharedReference</code> production.

1. Let |navigable| be [=get the navigable=] with |realm|.

1. If |navigable| is <code>null</code>, return [=error=] with [=error code=]
   [=no such node=].

   Note: This happens when the realm isn't a Window global.

1. Let |shared id| be the value of the <code>sharedId</code> field of
   |shared reference|.

1. Let |node| be result of [=trying=] to [=get a node=] with |session|,
   |navigable| and |shared id|.

1. If |node| is <code>null</code>, return [=error=] with [=error code=]
   [=no such node=].

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |realm|.

1. If |node|'s [=node document=]'s [=Document/origin=] is not [=same origin
   domain=] with |environment settings|'s [=environment settings object/origin=]
   then return [=error=] with [=error code=] [=no such node=].

   Note: This ensures that WebDriver-BiDi can not be used to pass objects
   between realms that do not otherwise permit script access.

1. Let |realm global object| be the [=realm/global object=] of the |realm|.

1. If the |realm global object| is {{SandboxWindowProxy}} object, set
   |node| to the {{SandboxProxy}} wrapping |node| in the |realm|.

1. Return [=success=] with data |node|.

</div>

#### The script.RemoteValue Type #### {#type-script-RemoteValue}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.RemoteValue = (
  script.PrimitiveProtocolValue /
  script.SymbolRemoteValue /
  script.ArrayRemoteValue /
  script.ObjectRemoteValue /
  script.FunctionRemoteValue /
  script.RegExpRemoteValue /
  script.DateRemoteValue /
  script.MapRemoteValue /
  script.SetRemoteValue /
  script.WeakMapRemoteValue /
  script.WeakSetRemoteValue /
  script.GeneratorRemoteValue /
  script.ErrorRemoteValue /
  script.ProxyRemoteValue /
  script.PromiseRemoteValue /
  script.TypedArrayRemoteValue /
  script.ArrayBufferRemoteValue /
  script.NodeListRemoteValue /
  script.HTMLCollectionRemoteValue /
  script.NodeRemoteValue /
  script.WindowProxyRemoteValue
)

script.ListRemoteValue = [*script.RemoteValue];

script.MappingRemoteValue = [*[(script.RemoteValue / text), script.RemoteValue]];

script.SymbolRemoteValue = {
  type: "symbol",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.ArrayRemoteValue = {
  type: "array",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.ListRemoteValue,
}

script.ObjectRemoteValue = {
  type: "object",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.MappingRemoteValue,
}

script.FunctionRemoteValue = {
  type: "function",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.RegExpRemoteValue = {
  script.RegExpLocalValue,
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.DateRemoteValue = {
  script.DateLocalValue,
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.MapRemoteValue = {
  type: "map",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.MappingRemoteValue,
}

script.SetRemoteValue = {
  type: "set",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.ListRemoteValue
}

script.WeakMapRemoteValue = {
  type: "weakmap",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.WeakSetRemoteValue = {
  type: "weakset",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.GeneratorRemoteValue = {
  type: "generator",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.ErrorRemoteValue = {
  type: "error",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.ProxyRemoteValue = {
  type: "proxy",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.PromiseRemoteValue = {
  type: "promise",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.TypedArrayRemoteValue = {
  type: "typedarray",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.ArrayBufferRemoteValue = {
  type: "arraybuffer",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
}

script.NodeListRemoteValue = {
  type: "nodelist",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.ListRemoteValue,
}

script.HTMLCollectionRemoteValue = {
  type: "htmlcollection",
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.ListRemoteValue,
}

script.NodeRemoteValue = {
  type: "node",
  ? sharedId: script.SharedId,
  ? handle: script.Handle,
  ? internalId: script.InternalId,
  ? value: script.NodeProperties,
}

script.NodeProperties = {
  nodeType: js-uint,
  childNodeCount: js-uint,
  ? attributes: {*text => text},
  ? children: [*script.NodeRemoteValue],
  ? localName: text,
  ? mode: "open" / "closed",
  ? namespaceURI: text,
  ? nodeValue: text,
  ? shadowRoot: script.NodeRemoteValue / null,
}

script.WindowProxyRemoteValue = {
  type: "window",
  value: script.WindowProxyProperties,
  ? handle: script.Handle,
  ? internalId: script.InternalId
}

script.WindowProxyProperties = {
  context: browsingContext.BrowsingContext
}
</pre>

Issue: Add WASM types?

Issue: Should WindowProxy get attributes in a similar style to Node?

Issue: handle String / Number / etc. wrapper objects specially?

Values accessible from the ECMAScript runtime are represented by a mirror
object, specified as <code>script.RemoteValue</code>. The value's type is
specified in the <code>type</code> property. In the case of JSON-representable
primitive values, this contains the value in the <code>value</code> property; in
the case of non-JSON-representable primitives, the <code>value</code> property
contains a string representation of the value.

For non-primitive objects, the <code>handle</code> property, when present,
contains a unique string handle to the object. The handle is unique for each
serialization. The remote end will keep objects with a corresponding handle
alive until such a time that <code>script.disown</code> is called with that
handle, or the realm itself is to be discarded (e.g. due to navigation).

For some non-primitive types, the <code>value</code> property contains a
representation of the data in the ECMAScript object; for container types this
can contain further <code>script.RemoteValue</code> instances. The
<code>value</code> property can be null or omitted if there is a duplicate
object i.e. the object has already been serialized in the current
<code>script.RemoteValue</code>, perhaps as part of a cycle, or otherwise when
the maximum serialization depth is reached.

In case of duplicated objects in the same <code>script.RemoteValue</code>, the
value is provided only for one of the remote values, while the
unique-per-ECMAScript-object <code>internalId</code> is provided for all the
duplicated objects for a given serialization.

[=Nodes=] are also represented by <code>script.RemoteValue</code>
instances. These have a partial serialization of the node in the value property.

Issue: reconsider mirror objects' lifecycle.

Note: mirror objects do not keep the original object alive in the runtime. If an
object is discarded in the runtime, subsequent attempts to access it via the
protocol will result in an error.

<div algorithm>
To get the <dfn>handle for an object</dfn> given |realm|, |ownership type| and
|object|:

1. If |ownership type| is equal "<code>none</code>", return <code>null</code>.

1. Let |handle id| be a new, unique, string handle for |object|.

1. Let |handle map| be |realm|'s [=handle object map=]

1. Set |handle map|[|handle id|] to |object|.

1. Return |handle id| as a result.

</div>

<div algorithm>
To <dfn>get shared id for a node</dfn> given |node|, and |session|:

1. Let |node| be [=unwrapped=] |node|.

1. If |node| does not implement {{Node}}, return null.

1. Let |navigable| be |node|'s [=/node navigable=].

1. If |navigable| is null, return null.

1. Return [=get or create a node reference=] with |session|, |navigable| and
   |node|.

</div>

<div algorithm>
To <dfn>set internal ids if needed</dfn> given |serialization internal map|,
|remote value| and |object|:

1. If the |serialization internal map| does not contain |object|, set
   |serialization internal map|[|object|] to |remote value|.

1. Otherwise, run the following steps:

    1. Let |previously serialized remote value| be
       |serialization internal map|[|object|].

    1. If |previously serialized remote value| does not have a field
       <code>internalId</code>, run the following steps:

      1. Let |internal id| be the string representation of a [[!RFC9562|UUID]]
         based on truly random, or pseudo-random numbers.

      1. Set the <code>internalId</code> field of
          |previously serialized remote value| to |internal id|.

    1. Set the <code>internalId</code> field of |remote value| to a field
       <code>internalId</code> in |previously serialized remote value|.

</div>

<div algorithm>

To <dfn>serialize as a remote value</dfn> given |value|, |serialization options|,
an |ownership type|, a |serialization internal map|, a |realm| and a |session|:

1. Let |remote value| be a result of [=serialize primitive protocol value=]
   given a |value|.

1. If |remote value| is not undefined, return |remote value|.

1. Let |handle id| be the [=handle for an object=] with |realm|, |ownership type|
   and |value|.

1. Set |ownership type| to "<code>none</code>".

1. Let |known object| be <code>true</code>, if |value| is in the
   |serialization internal map|, otherwise <code>false</code>.

1. In the following list of conditions and associated steps, run the first set
   of steps for which the associated condition is true:

  <dl>
    <dt>[=/Type=](|value|) is Symbol
    <dd>Let |remote value| be a [=/map=] matching the <code>script.SymbolRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>[=IsArray=](|value|)
    <dd>Let |remote value| be [=serialize an Array-like=] with |session|,
        <code>script.ArrayRemoteValue</code>, |handle id|, |known object|, |value|,
        |serialization options|, |ownership type|, |serialization internal map|,
        |realm|, and |session|.

    <dt>[=IsRegExp=](|value|)
    <dd>
    1. Let |pattern| be [=ToString=]([=Get=](|value|, "source")).

    1. Let |flags| be [=ToString=]([=Get=](|value|, "flags")).

    1. Let |serialized| be a [=/map=] matching the <code>script.RegExpValue</code> production in the
       {^local end definition^}, with the <code>pattern</code> property set to the
       |pattern| and the the <code>flags</code> property set to the |flags|.

    1. Let |remote value| be a [=/map=] matching the <code>script.RegExpRemoteValue</code>
       production in the {^local end definition^}, with the <code>handle</code>
       property set to |handle id| if it's not null, or omitted otherwise, and
       the <code>value</code> property set to |serialized|.

    <dt>|value| has a \[[DateValue]] [=internal slot=].
    <dd>
    1. Set |serialized| to [=Call=]([=Date.prototype.toISOString=], |value|).

    1. Assert: |serialized| is not a [=throw completion=].

    1. Let |remote value| be a [=/map=] matching the <code>script.DateRemoteValue</code>
       production in the {^local end definition^}, with the <code>handle</code>
       property set to |handle id| if it's not null, or omitted otherwise, and the
       value set to |serialized|.

    <dt>|value| has a \[[MapData]] [=internal slot=]
    <dd>

    1. Let |remote value| be a [=/map=] matching the <code>script.MapRemoteValue</code>
       production in the {^local end definition^}, with the
       <code>handle</code> property set to |handle id| if it's not null, or omitted
       otherwise.

    1. [=Set internal ids if needed=] with |serialization internal map|,
       |remote value| and |value|.

    1. Let |serialized| be null.

    1. If |known object| is <code>false</code>, and |serialization
       options|["<code>maxObjectDepth</code>"] is not 0, run the following
       steps:

       1. Let |serialized| be the result of [=serialize as a mapping=] with
          [=CreateMapIterator=](|value|, key+value), |serialization options|,
          |ownership type|, |serialization internal map|, |realm|, and |session|.

    1. If |serialized| is not null, set field <code>value</code> of |remote value| to
       |serialized|.

    <dt>|value| has a \[[SetData]] [=internal slot=]
    <dd>
    1. Let |remote value| be a [=/map=] matching the <code>script.SetRemoteValue</code>
       production in the {^local end definition^}, with the
       <code>handle</code> property set to |handle id| if it's not null, or omitted
       otherwise.

    1. [=Set internal ids if needed=] with |serialization internal map|,
       |remote value| and |value|.

    1. Let |serialized| be null.

    1. If |known object| is <code>false</code>, and |serialization
       options|["<code>maxObjectDepth</code>"] is not 0, run the following
       steps:

       1. Let |serialized| be the result of [=serialize as a list=] with
          [=CreateSetIterator=](|value|, value), |serialization options|,
          |ownership type|, |serialization internal map|, |realm|, and
          |session|.

    1. If |serialized| is not null, set field <code>value</code> of |remote value| to
       |serialized|.

    <dt>|value| has a \[[WeakMapData]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.WeakMapRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has a \[[WeakSetData]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.WeakSetRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has a \[[GeneratorState]] [=internal slot=] or \[[AsyncGeneratorState]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.GeneratorRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has an \[[ErrorData]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.ErrorRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has a \[[ProxyHandler]] [=internal slot=] and a \[[ProxyTarget]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.ProxyRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>[=IsPromise=](|value|)
    <dd>Let |remote value| be a [=/map=] matching the <code>script.PromiseRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has a \[[TypedArrayName]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.TypedArrayRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| has an \[[ArrayBufferData]] [=internal slot=]
    <dd>Let |remote value| be a [=/map=] matching the <code>script.ArrayBufferRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>|value| is a [=platform object=] that implements {{NodeList}}
    <dd>Let |remote value| be [=serialize an Array-like=] with
        <code>script.NodeListRemoteValue</code>,|handle id|, |known object|, |value|,
        |serialization options|, |ownership type|, |serialization internal map|,
        |realm|, and |session|.

    <dt>|value| is a [=platform object=] that implements {{HTMLCollection}}
    <dd>Let |remote value| be [=serialize an Array-like=] with
        <code>script.HTMLCollectionRemoteValue</code>, |handle id|, |known object|, |value|,
        |serialization options|, |ownership type|, |known object|, |serialization internal map|,
        |realm|, and |session|.

    <dt>|value| is a [=platform object=] that implements {{Node}}
    <dd>
      1. Let |shared id| be [=get shared id for a node=] with |value| and |session|.

      1. Let |remote value| be a [=/map=] matching the <code>script.NodeRemoteValue</code>
         production in the {^local end definition^}, with the <code>sharedId</code>
         property set to |shared id| if it's not null, or omitted otherwise, and the
         <code>handle</code> property set to |handle id| if it's not null, or omitted
         otherwise.

      1. [=Set internal ids if needed=] with |serialization internal map|,
         |remote value| and |value|.

      1. Let |serialized| be null.

      1. If |known object| is <code>false</code>, run the following steps:

          1. Let |serialized| be a [=/map=].

          1. Set |serialized|["<code>nodeType</code>"] to [=Get=](|value|, "nodeType").

          1. Set |node value| to [=Get=](|value|, "nodeValue").

          1. If |node value| is not null set
             |serialized|["<code>nodeValue</code>"] to |node value|.

          1. If |value| implements {{Element}} or {{Attr}}:

            1. Set |serialized|["<code>localName</code>"] to [=Get=](|value|, "localName").

            1. Set |serialized|["<code>namespaceURI</code>"] to [=Get=](|value|, "namespaceURI")

          1. Let |child node count| be the [=list/size=] of |value|'s <a spec=dom>children</a>.

          1. Set |serialized|["<code>childNodeCount</code>"] to |child node count|.

          1. If |serialization options|["<code>maxDomDepth</code>"] is equal to
             0, or if |value| implements {{ShadowRoot}} and |serialization
             options|["<code>includeShadowTree</code>"] is "<code>none</code>",
             or if |serialization options|["<code>includeShadowTree</code>"] is
             "<code>open</code>" and |value|'s <a spec=dom for=ShadowRoot>mode</a> is
             "<code>closed</code>", let |children| be null.

             Otherwise, let |children| be an empty
             [=/list=] and, for each node |child| in the <a spec=dom>children</a> of
             |value|:

            1. Let |child serialization options| be a [=map/clone=] of
               |serialization options|.

            1. If |child serialization options|["<code>maxDomDepth</code>"] is
               not null, set |child serialization
               options|["<code>maxDomDepth</code>"] to |child serialization
               options|["<code>maxDomDepth</code>"] - 1.

            1. Let |serialized| be the result of [=serialize as a remote value=]
               with |child|, |child serialization options|, |ownership type|,
               |serialization internal map|, |realm|, and |session|.

            1. Append |serialized| to |children|.

          1. If |children| is not null, set |serialized|["<code>children</code>"] to |children|.

          1. If |value| implements {{Element}}:

             1. Let |attributes| be a new [=/map=].

             1. For each |attribute| in |value|'s [=Element/attribute list=]:

               1. Let |name| be |attribute|'s [=Attr/qualified name=]

               1. Let |value| be |attribute|'s [=Attr/value=].

               1. Set |attributes|[|name|] to |value|

             1. Set |serialized|["<code>attributes</code>"] to |attributes|.

             1. Let |shadow root| be |value|'s [=Element/shadow root=].

             1. If |shadow root| is null, let |serialized shadow| be null.
                Otherwise run the following substeps:

               1. Let |serialized shadow| be the result of
                  [=serialize as a remote value=] with |shadow root|,
                  |serialization options|, |ownership type|, |serialization internal map|,
                  |realm|, and |session|.

             1. Set |serialized|["<code>shadowRoot</code>"] to |serialized shadow|.

          1. If |value| implements {{ShadowRoot}}, set |serialized|["<code>mode</code>"]
             to |value|'s <a spec=dom for=ShadowRoot>mode</a>.

      1. If |serialized| is not null, set field <code>value</code> of |remote value| to
         |serialized|.

    <dt>|value| is a [=platform object=] that implements {{WindowProxy}}
    <dd>
    1. Let |window| be the value of |value|'s \[[WindowProxy]] [=internal
       slot=].

    1. Let |navigable| be |window|'s [=window/navigable=].

    1. Let |navigable id| be the [=navigable id=] for |navigable|.

    1. Let |serialized| be a [=/map=] matching the
       <code>script.WindowProxyProperties</code> production in the {^local end
       definition^} with the <code>context</code> property set to |navigable id|.

    1. Let |remote value| be a [=/map=] matching the <code>script.WindowProxyRemoteValue</code>
       production in the {^local end definition^}, with the <code>handle</code>
       property set to |handle id| if it's not null, or omitted otherwise, and
       the <code>value</code> property set to |serialized|.

    <dt>|value| is a [=platform object=]
    <dd>1. Let |remote value| be a [=/map=] matching the <code>script.ObjectRemoteValue</code>
           production in the {^local end definition^}, with the <code>handle</code>
           property set to |handle id| if it's not null, or omitted otherwise.

    <dt>[=IsCallable=](|value|)
    <dd>Let |remote value| be a [=/map=] matching the <code>script.FunctionRemoteValue</code>
        production in the {^local end definition^}, with the <code>handle</code>
        property set to |handle id| if it's not null, or omitted otherwise.

    <dt>Otherwise:
    <dd>
    1. [=Assert=]: [=/Type=](|value|) is Object

    1. Let |remote value| be a [=/map=] matching the <code>script.ObjectRemoteValue</code> production
       in the {^local end definition^}, with the <code>handle</code> property set
       to |handle id| if it's not null, or omitted otherwise.

    1. [=Set internal ids if needed=] with |serialization internal map|,
       |remote value| and |value|.

    1. Let |serialized| be null.

    1. If |known object| is <code>false</code>, and |serialization
       options|["<code>maxObjectDepth</code>"] is not 0, run the following
       steps:

       1. Let |serialized| be the result of [=serialize as a mapping=] with
          [=EnumerableOwnPropertyNames=](|value|, key+value), |serialization options|,
          |ownership type|, |serialization internal map|,
          |realm|, and |session|.

    1. If |serialized| is not null, set field <code>value</code> of |remote value| to
       |serialized|.

  </dl>

1. Return |remote value|

Issue: |children| and child nodes are different things. Either <code>childNodeCount</code> should
reference to <code>childNodes</code>, or it should be renamed to <code>childrenCount</code>.

</div>

<div algorithm>
To <dfn>serialize an Array-like</dfn> given |production|, |handle id|, |known object|,
|value|, |serialization options|, |ownership type|, |serialization internal map|, |realm|,
and |session|:

1. Let |remote value| be a [=/map=] matching |production|, with the
   <code>handle</code> property set to |handle id| if it's not null, or omitted
   otherwise.

1. [=Set internal ids if needed=] with |serialization internal map|,
   |remote value| and |value|.

1. If |known object| is <code>false</code>, and |serialization
   options|["<code>maxObjectDepth</code>"] is not 0:

  1. Let |serialized| be the result of [=serialize as a list=] with
     [=CreateArrayIterator=](|value|, value), |serialization options|, |ownership type|,
     |serialization internal map|, |realm|, and |session|.

  1. If |serialized| is not null, set field <code>value</code> of |remote value| to
     |serialized|.

1. Return |remote value|

</div>

<div algorithm>
To <dfn>serialize as a list</dfn> given |iterable|, |serialization options|, |ownership type|,
|serialization internal map|, |realm|, and |session|:

1. If |serialization options|["<code>maxObjectDepth</code>"] is not null,
   assert: |serialization options|["<code>maxObjectDepth</code>"] is
   greater than 0.

1. Let |serialized| be a new list.

1. For each |child value| in [=IteratorToList=]([=GetIterator=](|iterable|, sync)):

  1. Let |child serialization options| be a [=map/clone=] of
     |serialization options|.

  1. If |child serialization options|["<code>maxObjectDepth</code>"] is
     not null, set |child serialization
     options|["<code>maxObjectDepth</code>"] to |child serialization
     options|["<code>maxObjectDepth</code>"] - 1.

  1. Let |serialized child| be the result of [=serialize as a remote value=]
     with |child value|, |child serialization options|, |ownership type|,
     |serialization internal map|, |realm|, and |session|.

  1. Append |serialized child| to |serialized|.

1. Return |serialized|

</div>

<div algorithm>

To <dfn>serialize as a mapping</dfn> given |iterable|, |serialization options|,
|ownership type|, |serialization internal map|, |realm|, and |session|:

1. If |serialization options|["<code>maxObjectDepth</code>"] is not null,
   assert: |serialization options|["<code>maxObjectDepth</code>"] is
   greater than 0.

1. Let |serialized| be a new list.

1. For |item| in [=IteratorToList=]([=GetIterator=](|iterable|, sync)):

  1. Assert: [=IsArray=](|item|)

  1. Let |property| be [=CreateListFromArrayLike=](|item|)

  1. Assert: |property| is a list of [=list/size=] 2

  1. Let |key| be |property|[0] and let |value| be |property|[1]

  1. Let |child serialization options| be a [=map/clone=] of
     |serialization options|.

  1. If |child serialization options|["<code>maxObjectDepth</code>"] is
     not null, set |child serialization
     options|["<code>maxObjectDepth</code>"] to |child serialization
     options|["<code>maxObjectDepth</code>"] - 1.

  1. If [=/Type=](|key|) is String, let |serialized key| be |child key|,
     otherwise let |serialized key| be the result of [=serialize as a remote value=]
     with |child key|, |child serialization options|, |ownership type|,
     |serialization internal map|, |realm|, and |session|.

  1. Let |serialized value| be the result of [=serialize as a remote value=]
     with |value|, |child serialization options|, |ownership type|,
     |serialization internal map|, |realm|, and |session|.

  1. Let |serialized child| be («|serialized key|, |serialized value|»).

  1. Append |serialized child| to |serialized|.

1. Return |serialized|

</div>

#### The script.ResultOwnership Type #### {#type-script-ResultOwnership}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.ResultOwnership = "root" / "none"
</pre>

The <code>script.ResultOwnership</code> specifies how the serialized value
ownership will be treated.

#### The script.SerializationOptions Type #### {#type-script-SerializationOptions}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.SerializationOptions = {
  ? maxDomDepth: (js-uint / null) .default 0,
  ? maxObjectDepth: (js-uint / null) .default null,
  ? includeShadowTree: ("none" / "open" / "all") .default "none",
}
</pre>

The <code>script.SerializationOptions</code> allows specifying how ECMAScript
objects will be serialized.

#### The script.SharedId Type #### {#type-script-SharedId}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.SharedId = text;
</pre>

The <code>script.SharedId</code> type represents a reference to a DOM {{Node}} that
is usable in any realm (including <a href=#sandbox-realm>Sandbox Realms</a>).

#### The script.StackFrame Type #### {#type-script-StackFrame}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.StackFrame = {
  columnNumber: js-uint,
  functionName: text,
  lineNumber: js-uint,
  url: text,
}
</pre>

A frame in a stack trace is represented by a <code>StackFrame</code>
object. This has a <code>url</code> property, which represents the URL of the
script, a <code>functionName</code> property which represents the name of the
executing function, and <code>lineNumber</code> and <code>columnNumber</code>
properties, which represent the line and column number of the executed code.

#### The script.StackTrace Type #### {#type-script-StackTrace}

{^Remote end definition^} and {^local end definition^}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
script.StackTrace = {
  callFrames: [*script.StackFrame],
}
</pre>

The <code>script.StackTrace</code> type represents the javascript stack at a point in
script execution.

Note: The details of how to get a list of stack frames, and the properties of
that list are underspecified, and therefore the details here are implementation
defined.

It is assumed that an implementation is able to generate a <dfn>list of stack
frames</dfn>, which is a list with one entry for each item in the javascript
call stack, starting from the most recent. Each entry is a single <dfn>stack
frame</dfn> corresponding to execution of a statement or expression in a script
|script|, which contains the following fields:

<dl>
 <dt><dfn for=stackframe>script url</dfn>
 <dd>The url of the resource containing |script|
 <dt><dfn for=stackframe>function</dfn>
 <dd>The name of the function being executed
 <dt><dfn for=stackframe>line number</dfn>
 <dd>The zero-based line number of the executed code, relative to the top
 of the resource containing |script|.
 <dt><dfn for=stackframe>column number</dfn>
 <dd>The zero-based column number of the executed code, relative to the start of
 the line in the resource containing |script|.
</dl>

<div algorithm>

To <dfn>construct a stack trace</dfn>, with a list of stack frames |stack|:

1. Let |call frames| be a new list.

1. For each [=stack frame=] |frame| in |stack|,
   starting from the most recently executed frame, run the following steps:

   1. Let |url| be the result of running the [=URL serializer=], given
      the <a spec=url for=/>URL</a> of |frame|'s [=stackframe/script url=].

   1. Let |frame info| be a new [=/map=] matching the <code>script.StackFrame</code>
      production, with the <code>url</code> field set to |url|, the
      <code>functionName</code> field set to |frame|'s [=stackframe/function=],
      the <code>lineNumber</code> field set to |frame|'s [=stackframe/line
      number=] and the <code>columnNumber</code> field set to |frame|'s
      [=stackframe/column number=].

1. Append |frame info| to |call frames|.

1. Let |stack trace| be a new [=/map=] matching the <code>script.StackTrace</code>
   production, with the <code>callFrames</code> property set to |call frames|.

1. Return |stack trace|.

</div>

The <dfn>current stack trace</dfn> is the result of [=construct a stack trace=]
given a [=list of stack frames=] representing the callstack of the [=running
execution context=].

<div algorithm>

The <dfn>stack trace for an exception</dfn> with an exception, or a [=Completion
Record=] of type <code>throw</code>, |exception|, is given by:

1. If |exception| is a value that has been thrown as an exception, let |record|
   be the [=Completion Record=] created to throw |exception|. Otherwise let
   |record| be |exception|.

1. Let |stack| be the [=list of stack frames=] corresponding to execution at the
   point |record| was created.

1. Return [=construct a stack trace=] given |stack|.

</div>

#### The script.Source Type #### {#type-script-Source}

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
script.Source = {
  realm: script.Realm,
  ? context: browsingContext.BrowsingContext,
  ? userContext: browser.UserContext
}
</pre>

The <code>script.Source</code> type represents a <code>script.Realm</code> with
an optional <code>browsingContext.BrowsingContext</code> and related
<code>browser.UserContext</code> in which a script related event occurred.

<div algorithm>
To <dfn>get the source</dfn> given |source realm|:

1. Let |realm| be the [=realm id=] for |source realm|.

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |source realm|.

1. If |environment settings| has a <a>associated <code>Document</code></a>:

  1. Let |document| be environment settings’ <a>associated <code>Document</code></a>.

  1. Let |navigable| be |document|’s [=/node navigable=].

  1. Let |navigable id| be the [=navigable id=] for |navigable| if |navigable| is not null.

  1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

  Otherwise let |navigable| be null.

1. Let |source| be a [=/map=] matching the
   <code>script.Source</code> production with the <code>realm</code>
   field set to |realm|, the <code>context</code> field set
   to |navigable id| if |navigable| is not null, or unset otherwise,
   and the <code>userContext</code> field set to |user context id| if
   |navigable is not null, or unset otherwise.

1. Return |source|.

</div>

#### The script.Target Type #### {#type-script-Target}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
script.RealmTarget = {
  realm: script.Realm
}

script.ContextTarget = {
  context: browsingContext.BrowsingContext,
  ? sandbox: text
}

script.Target = (
  script.ContextTarget /
  script.RealmTarget
)
</pre>

The <code>script.Target</code> type represents a value that is either a
<code>script.Realm</code> or a <code>browsingContext.BrowsingContext</code>.
This is useful in cases where a navigable identifier can stand in for the realm
associated with the navigable's active document.

<div algorithm>
To <dfn>get a realm from a navigable</dfn> given |navigable id| and |sandbox|:

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. If |sandbox| is null or is an empty string:

  1. Let |document| be |navigable|'s [=active document=].

  1. Let |environment settings| be the [=environment settings object=] whose
     [=relevant global object=]'s <a>associated <code>Document</code></a> is
     |document|.

  1. Let |realm| be |environment settings|' [=realm execution context=]'s
     Realm component.

1. Otherwise: let |realm| be result of [=trying=] to
   [=get or create a sandbox realm=] given |sandbox| and
   |navigable|.

1. Return [=success=] with data |realm|

Issue: This has the wrong error code
</div>

<div algorithm>
To <dfn>get a realm from a target</dfn> given |target|:

1. If |target| matches the <code>script.ContextTarget</code> production:

  1. Let |sandbox| be null.

  1. If |target| [=map/contains=] "<code>sandbox</code>", set |sandbox| to
     |target|["<code>sandbox</code>"].

  1. Let |realm| be [=get a realm from a navigable=] with
     |target|["<code>context</code>"] and |sandbox|.

1. Otherwise:

  1. Assert: |target| matches the <code>script.RealmTarget</code> production.

  1. Let |realm id| be the value of the <code>realm</code> field of |target|.

  1. Let |realm| be [=get a realm=] given |realm id|.

1. Return [=success=] with data |realm|

Issue: This has the wrong error code
</div>

### Commands ### {#module-script-commands}

#### The script.addPreloadScript Command ####  {#command-script-addPreloadScript}

The <dfn export for=commands>script.addPreloadScript</dfn> command adds a [=preload
script=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      script.AddPreloadScript = (
        method: "script.addPreloadScript",
        params: script.AddPreloadScriptParameters
      )

      script.AddPreloadScriptParameters = {
        functionDeclaration: text,
        ? arguments: [*script.ChannelValue],
        ? contexts: [+browsingContext.BrowsingContext],
        ? userContexts: [+browser.UserContext],
        ? sandbox: text
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      script.AddPreloadScriptResult = {
        script: script.PreloadScript
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for script.addPreloadScript">
The [=remote end steps=] given |session| and |command parameters| are:

1. If |command parameters| [=map/contains=] "<code>userContexts</code>"
   and |command parameters| [=map/contains=] "<code>contexts</code>",
   return [=error=] with [=error code=] [=invalid argument=].

1. Let |function declaration| be the <code>functionDeclaration</code> field of |command
   parameters|.

1. Let |arguments| be the <code>arguments</code> field of |command
   parameters| if present, or an empty [=/list=] otherwise.

1. Let |user contexts| to be a [=/set=].

1. Let |navigables| be null.

1. If the <code>contexts</code> field of |command parameters| is present:

   1. Set |navigables| to an empty [=/set=].

   1. For each |navigable id| of |command parameters|["<code>contexts</code>"]

      1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with |navigable id|.

      1. If |navigable| is not a [=/top-level traversable=], return [=error=] with [=error code=] [=invalid argument=].

      1. Append |navigable| to |navigables|.

1. Otherwise, if |command parameters| [=map/contains=] <code>userContexts</code>:

  1. Set |user contexts| to [=set/create|create a set=] with |command parameters|["<code>userContexts</code>"].

  1. For each |user context id| of |user contexts|:

    1. Set |user context| to [=get user context=] with |user context id|.

    1. If |user context| is null, return [=error=] with [=error code=] [=no such user context=].

1. Let |sandbox| be the value of the "<code>sandbox</code>" field in |command
   parameters|, if present, or null otherwise.

1. Let |script| be the string representation of a [[!RFC9562|UUID]].

1. Let |preload script map| be |session|'s [=preload script map=].

1. Set |preload script map|[|script|] to a struct with
   <code>function declaration</code> |function declaration|,
   <code>arguments</code> |arguments|,
   <code>contexts</code> |navigables|,
   <code>sandbox</code> |sandbox|, and
   <code>user contexts</code> |user contexts|.

1. Return a new [=/map=] matching the <code>script.AddPreloadScriptResult</code> with the
   <code>script</code> field set to |script|.

</div>

#### The script.disown Command ####  {#command-script-disown}

The <dfn export for=commands>script.disown</dfn> command disowns the given handles.
This does not guarantee the handled object will be garbage collected, as there can be
other handles or strong ECMAScript references.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      script.Disown = (
        method: "script.disown",
        params: script.DisownParameters
      )

      script.DisownParameters = {
        handles: [*script.Handle]
        target: script.Target;
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      script.DisownResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for script.disown">
The [=remote end steps=] with |command parameters| are:

1. Let |realm| be the result of [=trying=] to [=get a realm from a target=]
   given the value of the <code>target</code> field of |command parameters|.

1. Let |handles| the value of the <code>handles</code> field of |command parameters|.

1. For each |handle id| of |handles|:

   1. Let |handle map| be |realm|'s [=handle object map=]

   1. If |handle map| contains |handle id|, remove |handle id| from the |handle map|.

1. Return [=success=] with data null.

</div>

#### The script.callFunction Command ####  {#command-script-callFunction}

The <dfn export for=commands>script.callFunction</dfn> command calls a provided
function with given arguments in a given realm.

<code>RealmInfo</code> can be either a realm or a navigable.

Note: In case of an arrow function in <code>functionDeclaration</code>, the
<code>this</code> argument doesn't affect function's <code>this</code> binding.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      script.CallFunction = (
        method: "script.callFunction",
        params: script.CallFunctionParameters
      )

      script.CallFunctionParameters = {
        functionDeclaration: text,
        awaitPromise: bool,
        target: script.Target,
        ? arguments: [*script.LocalValue],
        ? resultOwnership: script.ResultOwnership,
        ? serializationOptions: script.SerializationOptions,
        ? this: script.LocalValue,
        ? userActivation: bool .default false,
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      script.CallFunctionResult = script.EvaluateResult
    </pre>
   </dd>
</dl>

Issue: TODO: Add timeout argument as described in the script.evaluate.

<div algorithm>

To <dfn>deserialize arguments</dfn> with given |realm|, |serialized arguments list|
and |session|:

1. Let |deserialized arguments list| be an empty [=/list=].

1. For each |serialized argument| of |serialized arguments list|:

  1. Let |deserialized argument| be the result of [=trying=] to [=deserialize local value=]
     given |serialized argument|, |realm| and |session|.

  1. Append |deserialized argument| to the |deserialized arguments list|.

1. Return [=success=] with data |deserialized arguments list|.

</div>

<div algorithm>
To <dfn>evaluate function body</dfn> given |function declaration|,
|environment settings|, |base URL|, and |options|:

Note: the |function declaration| is parenthesized and evaluated.

1. Let |bypassDisabledScripting| be true.

1. Let |parenthesized function declaration| be [=concatenate=]
   «"<code>(</code>", |function declaration|, "<code>)</code>"»

1. Let |function script| be the result of [=create a classic script=] with
   |parenthesized function declaration|, |environment settings|, |base URL|,
   |options| and |bypassDisabledScripting|.

1. [=Prepare to run script=] with |environment settings|.

1. Let |function body evaluation status| be [=ScriptEvaluation=](|function script|'s record).

1. [=Clean up after running script=] with |environment settings|.

1. Return |function body evaluation status|.

</div>


<div algorithm="remote end steps for script.callFunction">

The [=remote end steps=] with |session| and |command parameters| are:

1. Let |realm| be the result of [=trying=] to [=get a realm from a target=]
   given the value of the <code>target</code> field of |command parameters|.

1. Let |realm id| be |realm|'s [=realm id=].

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |realm|.

1. Let |command arguments| be the value of the <code>arguments</code> field
   of |command parameters|.

1. Let |deserialized arguments| be an empty [=/list=].

1. If |command arguments| is not null, set |deserialized arguments| to the result of [=trying=] to
   [=deserialize arguments=] given |realm|, |command arguments| and |session|.

1. Let |this parameter| be the value of the <code>this</code> field
   of |command parameters|.

1. Let |this object| be null.

1. If |this parameter| is not null, set |this object| to the result of [=trying=] to
   [=deserialize local value=] given |this parameter|, |realm| and |session|.

1. Let |function declaration| be the value of the
   <code>functionDeclaration</code> field of |command parameters|.

1. Let |await promise| be the value of the <code>awaitPromise</code> field of
   |command parameters|.

1. Let |serialization options| be the value of the
   <code>serializationOptions</code> field of |command parameters|, if present,
   or otherwise a [=/map=] matching the <code>script.SerializationOptions</code>
   production with the fields set to their default values.

1. Let |result ownership| be the value of the <code>resultOwnership</code> field of
   |command parameters|, if present, or <code>none</code> otherwise.

1. Let |base URL| be the [=API base URL=] of |environment settings|.

1. Let |options| be the [=default script fetch options=].

1. Let |function body evaluation status| be the
   result of [=evaluate function body=] with |function declaration|,
   |environment settings|, |base URL|, and |options|.

1. If |function body evaluation status|.\[[Type]] is <code>throw</code>:

   1. Let |exception details| be the result of [=get exception details=] given
      |realm|, |function body evaluation status|, |result ownership| and |session|.

   1. Return a new [=/map=] matching the <code>script.EvaluateResultException</code>
      production, with the <code>exceptionDetails</code> field set to
      |exception details|.

1. Let |function object| be |function body evaluation status|.\[[Value]].

1. If [=IsCallable=](|function object|) is <code>false</code>:

   1. Return an [=error=] with [=error code=] [=invalid argument=]

1. If |command parameters|["<code>userActivation</code>"] is true, run [=activation notification=] steps.

1. [=Prepare to run script=] with |environment settings|.

1. Set |evaluation status| to
   [=Call=](|function object|, |this object|, |deserialized arguments|).

1. If |evaluation status|.\[[Type]] is <code>normal</code>, and |await promise| is
   <code>true</code>, and [=IsPromise=](|evaluation status|.\[[Value]]):

   1. Set |evaluation status| to
      <a spec=ECMASCRIPT>Await</a>(|evaluation status|.\[[Value]]).

1. [=Clean up after running script=] with |environment settings|.

1. If |evaluation status|.\[[Type]] is <code>throw</code>:

  1. Let |exception details| be the result of [=get exception details=] given
     |realm|, |evaluation status|, |result ownership| and |session|.

  1. Return a new [=/map=] matching the <code>script.EvaluateResultException</code>
     production, with the <code>exceptionDetails</code> field set to
     |exception details|.

1. Assert: |evaluation status|.\[[Type]] is <code>normal</code>.

1. Let |result| be the result of [=serialize as a remote value=] with
   |evaluation status|.\[[Value]], |serialization options|, |result ownership|,
   a new [=/map=] as serialization internal map, |realm| and |session|.

1. Return a new [=/map=] matching the <code>script.EvaluateResultSuccess</code>
   production, with the <code>realm</code> field set to |realm id|,
   and the <code>result</code> field set to |result|.

</div>

#### The script.evaluate Command ####  {#command-script-evaluate}

The <dfn export for=commands>script.evaluate</dfn> command evaluates a provided
script in a given realm. For convenience a navigable can be provided in
place of a realm, in which case the realm used is the realm of the browsing
context's active document.

The method returns the value of executing the provided script, unless it returns
a promise and <code>awaitPromise</code> is true, in which case the resolved value of
the promise is returned.

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      script.Evaluate = (
        method: "script.evaluate",
        params: script.EvaluateParameters
      )

      script.EvaluateParameters = {
        expression: text,
        target: script.Target,
        awaitPromise: bool,
        ? resultOwnership: script.ResultOwnership,
        ? serializationOptions: script.SerializationOptions,
        ? userActivation: bool .default false,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <!--There is a duplication between command's result and type-->
    <code>
      script.EvaluateResult
    </code>
   </dd>
</dl>

TODO: Add timeout argument. It's not totally clear how this ought to work; in
Chrome it seems like the timeout doesn't apply to the promise resolve step, but
that likely isn't what clients want.

<div algorithm="remote end steps for script.evaluate">

The [=remote end steps=] given |session| and |command parameters| are:

1. Let |realm| be the result of [=trying=] to [=get a realm from a target=]
   given the value of the <code>target</code> field of |command parameters|.

1. Let |realm id| be |realm|'s [=realm id=].

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |realm|.

1. Let |source| be the value of the <code>expression</code> field of |command
   parameters|.

1. Let |await promise| be the value of the <code>awaitPromise</code> field of
   |command parameters|.

1. Let |serialization options| be the value of the
   <code>serializationOptions</code> field of |command parameters|, if present,
   or otherwise a [=/map=] matching the <code>script.SerializationOptions</code>
   production with the fields set to their default values.

1. Let |result ownership| be the value of the <code>resultOwnership</code> field of
   |command parameters|, if present, or <code>none</code> otherwise.

1. Let |options| be the [=default script fetch options=].

1. Let |base URL| be the [=API base URL=] of |environment settings|.

1. Let |bypassDisabledScripting| be true.

1. Let |script| be the result of [=create a classic script=] with |source|,
   |environment settings|, |base URL|, |options| and |bypassDisabledScripting|.

1. If |command parameters|["<code>userActivation</code>"] is true, run [=activation notification=] steps.

1. [=Prepare to run script=] with |environment settings|.

1. Set |evaluation status| to [=ScriptEvaluation=](|script|'s record).

1. If |evaluation status|.\[[Type]] is <code>normal</code>, |await promise| is
   true, and [=IsPromise=](|evaluation status|.\[[Value]]):

   1. Set |evaluation status| to <a spec=ECMASCRIPT>Await</a>(|evaluation status|.\[[Value]]).

1. [=Clean up after running script=] with |environment settings|.

1. If |evaluation status|.\[[Type]] is <code>throw</code>:

  1. Let |exception details| be the result of [=get exception details=] with
     |realm|, |evaluation status|, |result ownership| and |session|.

  1. Return a new [=/map=] matching the <code>script.EvaluateResultException</code>
     production, with the <code>realm</code> field set to |realm id|, and the
     <code>exceptionDetails</code> field set to |exception details|.

1. Assert: |evaluation status|.\[[Type]] is <code>normal</code>.

1. Let |result| be the result of [=serialize as a remote value=] with
   |evaluation status|.\[[Value]], |serialization options|, |result ownership|,
   a new [=/map=] as serialization internal map, |realm| and |session|.

1. Return a new [=/map=] matching the <code>script.EvaluateResultSuccess</code>
   production, with the with the <code>realm</code> field set to |realm id|, and
   the <code>result</code> field set to |result|.

</div>

#### The script.getRealms Command ####  {#command-script-getRealms}

The <dfn export for=commands>script.getRealms</dfn> command returns a list of
all realms, optionally filtered to [=realms=] of a specific type, or to the
realm associated with a [=/navigable=]'s [=active document=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      script.GetRealms = (
        method: "script.getRealms",
        params: script.GetRealmsParameters
      )

      script.GetRealmsParameters = {
        ? context: browsingContext.BrowsingContext,
        ? type: script.RealmType,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      script.GetRealmsResult = {
        realms: [*script.RealmInfo]
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for script.getRealms">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |environment settings| be a [=/list=] of all the [=environment settings objects=]
   that have their [=execution ready flag=] set.

1. If |command parameters| contains <code>context</code>:

  1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
     with |command parameters|["<code>context</code>"].

  1. Let |document| be |navigable|'s [=active document=].

  1. Let |navigable environment settings| be a [=/list=].

  1. For each |settings| of |environment settings|:

    1. If any of the following conditions hold:

      * The <a>associated <code>Document</code></a> of |settings|'
        [=relevant global object=] is |document|

      * The [=realm/global object=] specified by |settings| is a
        {{WorkerGlobalScope}} with |document| in its [=owner set=]

      Append |settings| to |navigable environment settings|.

  1. Set |environment settings| to |navigable environment settings|.

1. Let |realms| be a list.

1. For each |settings| of |environment settings|:

  1. Let |realm info| be the result of [=get the realm info=] given |settings|.

  1. If |command parameters| contains <code>type</code> and |realm
     info|["<code>type</code>"] is not equal to |command
     parameters|["<code>type</code>"] then [=continue=].

  1. If |realm info| is not null, append |realm info| to |realms|.

1. Let |body| be a [=/map=] matching the <code>script.GetRealmsResult</code> production,
   with the <code>realms</code> field set to |realms|.

1. Return [=success=] with data |body|.


Issue: Extend this to also allow realm parents e.g. for nested workers? Or get all ancestor workers.

Issue: We might want to have a more sophisticated filter system than just a
       literal match.

</div>

#### The script.removePreloadScript Command ####  {#command-script-removePreloadScript}

The <dfn export for=commands>script.removePreloadScript</dfn> command removes a
[=preload script=].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      script.RemovePreloadScript = (
        method: "script.removePreloadScript",
        params: script.RemovePreloadScriptParameters
      )

      script.RemovePreloadScriptParameters = {
        script: script.PreloadScript
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      script.RemovePreloadScriptResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for script.removePreloadScript">
The [=remote end steps=] given |session| and |command parameters| are:

1. Let |script| be the value of the "<code>script</code>" field in |command
   parameters|.

1. Let |preload script map| be |session|'s [=preload script map=].

1. If |preload script map| does not <a for=map>contain</a> |script|, return
   [=error=] with [=error code=] [=no such script=].

1. <a for=set>Remove</a> |script| from |preload script map|.

1. Return null

</div>


### Events ### {#module-script-events}

#### The script.message Event #### {#event-script-message}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        script.Message = (
         method: "script.message",
         params: script.MessageParameters
        )

       script.MessageParameters = {
         channel: script.Channel,
         data: script.RemoteValue,
         source: script.Source,
       }
      </pre>
   </dd>
</dl>

<div algorithm="remote end event trigger for script.message">
The [=remote end event trigger=] is the <dfn>emit a script message</dfn> steps,
given |session|, |realm|, |channel properties|, and |message|:

1. Let |environment settings| be the [=environment settings object=] whose
   [=realm execution context=]'s Realm component is |realm|.

1. Let |related navigables| be the result of [=get related navigables=] given |environment settings|.

1. If [=event is enabled=] given |session|, "<code>script.message</code>"
   and |related navigables|:

  1. If |channel properties| [=map/contains=]
     "<code>serializationOptions</code>", let |serialization options| be the
     value of the <code>serializationOptions</code> field of |channel
     properties|. Otherwise let |serialization options| be a [=/map=]
     matching the <code>script.SerializationOptions</code> production with the
     fields set to their default values.

  1. Let if |channel properties| [=map/contains=] "<code>ownership</code>", let
     |ownership type| be |channel properties|["<code>ownership</code>"]. Otherwise let
     |ownership type| be "<code>none</code>".

  1. Let |data| be the result of [=serialize as a remote value=] given
     |message|, |serialization options|, |ownership type|, a new [=/map=] as serialization
     internal map and |realm|.

  1. Let |source| be the [=get the source=] with |realm|.

  1. Let |params| be a [=/map=] matching the
     <code>script.MessageParameters</code> production, with the
     <code>channel</code> field set to |channel properties|["<code>channel</code>"],
     the <code>data</code> field set to |data|, and the <code>source</code>
     field set to |source|.

  1. Let |body| be a [=/map=] matching the <code>script.Message</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

</div>

#### The script.realmCreated Event #### {#event-script-realmCreated}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        script.RealmCreated = (
         method: "script.realmCreated",
         params: script.RealmInfo
        )
      </pre>
   </dd>
</dl>
The [=remote end event trigger=] is:

<div algorithm="remote end event trigger for script.realmCreated">

When any of the [=set up a window environment settings object=], [=set up a
worker environment settings object=] or [=set up a worklet environment settings
object=] algorithms are invoked, immediately prior to returning the settings
object:

1. Let |environment settings| be the newly created [=environment settings
   object=].

1. Let |realm info| be the result of [=get the realm info=] given
   |environment settings|.

1. If |realm info| is null, return.

1. Let |related navigables| be the result of [=get related navigables=] given |environment settings|.

1. Let |body| be a [=/map=] matching the <code>script.RealmCreated</code>
   production, with the <code>params</code> field set to |realm info|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>script.realmCreated</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

</div>

<div algorithm="remote end subscribe steps for script.realmCreated">

The [=remote end subscribe steps=] with [=subscribe priority=] 2, given
|session|, |navigables| and |include global| are:

1. Let |environment settings| be a list of all the [=environment settings objects=]
   that have their [=execution ready flag=] set.

1. For each |settings| of |environment settings|:

  1. Let |related navigables| be a new [=/set=].

  1. If the <a>associated <code>Document</code></a> of |settings|'
     [=relevant global object=] is a [=/Document=]:

     1. Let |navigable| be |settings|'s [=relevant global object=]'s
        <a>associated <code>Document</code></a>'s
        [=/node navigable=].

     1. If |navigable| is null, continue.

     1. Let |top-level traversible| be |navigable|'s [=navigable/top-level traversable=].

     1. If |top-level traversible| is not in |navigables|, continue.

     1. Append |top-level traversible| to |related navigables|.

    Otherwise, if |include global| is false, continue.

  1. Let |realm info| be the result of [=get the realm info=] given |settings|.

  1. If |realm info| is null, continue.

  1. Let |body| be a [=/map=] matching the <code>script.RealmCreated</code>
     production, with the <code>params</code> field set to |realm info|.

  1. If [=event is enabled=] given |session|,
     "<code>script.realmCreated</code>" and |related navigables|:

    1. [=Emit an event=] with |session| and |body|.

Issue: Should the order here be better defined?

</div>

#### The script.realmDestroyed Event #### {#event-script-realmDestroyed}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
       script.RealmDestroyed = (
         method: "script.realmDestroyed",
         params: script.RealmDestroyedParameters
       )

       script.RealmDestroyedParameters = {
         realm: script.Realm
       }

      </pre>
   </dd>
</dl>
The [=remote end event trigger=] is:

<div algorithm="remote end event trigger for script.realmDestroyed">
Define the following [=unloading document cleanup steps=] with |document|:

1. Let |related navigables| be an empty [=/set=].

1. Append |document|'s [=/navigable=] to |related navigables|.

1. For each |worklet global scope| in |document|'s [=worklet global scopes=]:

  1. Let |realm| be |worklet global scope|'s [=relevant Realm=].

  1. Let |realm id| be the [=realm id=] for |realm|.

  1. Let |params| be a [=/map=] matching the <code>script.RealmDestroyedParameters</code>
     production, with the <code>realm</code> field set of |realm id|.

  1. Let |body| be a [=/map=] matching the <code>script.RealmDestroyed</code>
     production, with the <code>params</code> field set to |params|.

  1. For each |session| in the [=set of sessions for which an event is enabled=]
     given "<code>script.realmDestroyed</code>" and |related navigables|:

    1. [=Emit an event=] with |session| and |body|.

1. Let |environment settings| be the [=environment settings object=] whose
   [=relevant global object=]'s <a>associated <code>Document</code></a> is
   |document|.


1. Let |realm| be |environment settings|' [=realm execution context=]'s Realm component.

1. Let |realm id| be the [=realm id=] for |realm|.

1. Let |params| be a [=/map=] matching the
   <code>script.RealmDestroyedParameters</code> production, with the
   <code>realm</code> field set to |realm id|.

1. Let |body| be a [=/map=] matching the <code>script.RealmDestroyed</code>
   production, with the <code>params</code> field set to |params|.

1. For each |session| in the [=set of sessions for which an event is enabled=]
   given "<code>script.realmDestroyed</code>" and |related navigables|:

  1. [=Emit an event=] with |session| and |body|.

Whenever a [=worker event loop=] |event loop| is destroyed, either because the
worker comes to the end of its lifecycle, or prematurely via the [=terminate a
worker=] algorithm:

1. Let |environment settings| be the [=environment settings object=] for which
   |event loop| is the [=responsible event loop=].

1. Let |related navigables| be the result of [=get related navigables=] given |environment settings|.

1. Let |realm| be |environment settings|'s [=environment settings object's Realm=].

1. Let |realm id| be the [=realm id=] for |realm|.

1. Let |params| be a [=/map=] matching the <code>script.RealmDestroyedParameters</code>
   production, with the <code>realm</code> field set of |realm id|.

1. Let |body| be a [=/map=] matching the <code>script.RealmDestroyed</code>
   production, with the <code>params</code> field set to |params|.

</div>

## The storage Module ## {#module-storage}

The <dfn export for=modules>storage</dfn> module contains functionality and
events related to storage.

A <dfn>storage partition</dfn> is a namespace within which the user agent may
organize persistent data such as [=cookies=] and local storage.

A <dfn>storage partition key</dfn> is a [=/map=] which uniquely identifies a [=storage partition=].

### Definition ### {#module-storage-definition}

{^Remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
StorageCommand = (
  storage.DeleteCookies //
  storage.GetCookies //
  storage.SetCookie
)
</pre>

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
StorageResult = (
  storage.DeleteCookiesResult /
  storage.GetCookiesResult /
  storage.SetCookieResult
)
</pre>

### Types ### {#module-storage-types}

#### The storage.PartitionKey Type #### {#type-storage-PartitionKey}

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl,remote-cddl">
storage.PartitionKey = {
  ? userContext: text,
  ? sourceOrigin: text,
  Extensible,
}
</pre>

The <code>storage.PartitionKey</code> type represents a [=storage partition key=].

The following <dfn>table of standard storage partition key attributes</dfn>
enumerates attributes with well-known meanings which a [=remote end=] may
choose to support. An implementation may define additional [=extension storage
partition key attributes=].

<table class="respec-simple">
  <tr>
    <th>Attribute
    <th>Definition
  <tr>
    <td>"<code>userContext</code>"
    <td>A [=user context id=]
  <tr>
    <td>"<code>sourceOrigin"</code>
    <td>The [=serialization of an origin|serialization of the origin=] of resources that can access the storage partition
</table>

[=Remote ends=] may support any number of <dfn>extension storage partition key
attributes</dfn>. In order to avoid conflicts with other implementations, these
attributes must begin with a unique identifier for the vendor and user-agent
followed by U+003A (:).

A [=remote end=] has a [=/map=] of <dfn>default values for storage partition
key attributes</dfn> which contains zero or more entries. Each key must be a
member of the [=table of standard storage partition key attributes=] where the
[=storage partition key=] corresponds to a standard storage partition, or an
[=extension storage partition key attribute=] where it does not, and the values
represent the default value of that partition key that will be used when the user
doesn't provide an explicit value. The precise entries are
implementation-defined and are determined by the storage partitioning adopted
by the implementation.

A [=remote end=] has a [=/list=] of <dfn>required partition key
attributes</dfn> which contains zero or more entries. Each key must be a member
of the [=table of standard storage partition key attributes=] where the
[=storage partition key=] corresponds to a standard storage partition, or an
[=extension storage partition key attribute=] where it does not. The precise
entries are implementation-defined and are determined by the storage
partitioning adopted by the implementation. This list includes only partition
keys for which no default is available. As such the list must not share any
entries with the keys of [=default values for storage partition key
attributes=].

<div algorithm>
To <dfn>deserialize filter</dfn> given |filter|:

1. Let |deserialized filter| to be an empty [=/map=].

1. For each |name| → |value| in |filter|:

  1. Let |deserialized name| be the field name corresponding to the JSON key |name| in
    the [=table for cookie conversion=].

  1. If |name| is "<code>value</code>", set |deserialized value| to [=deserialize protocol bytes=] with |value|,
     otherwise let |deserialized value| be |value|.

  1. [=map/Set=] |deserialized filter|[|deserialized name|] to |deserialized value|.

1. Return |deserialized filter|.

</div>

<div algorithm>
To <dfn>expand a storage partition spec</dfn> given |partition spec|:

1. If |partition spec| is null:

  1. Set |partition spec| to an empty [=/map=].

1. Otherwise, if |partition spec|["<code>type</code>"] is "<code>context</code>":

  1. Let |navigable| be the result of [=trying=] to [=get a navigable=] given |partition spec|["<code>context</code>"].

  1. Let |partition key| be the [=storage partition key|key=] of |navigable|'s [=associated storage partition=].

  1. Return [=success=] with data |partition key|.

1. Let |partition key| be an empty [=/map=].

1. For each |name| → |default value| in the [=default values for storage partition key attributes=]:

  1. Let |value| be |partition spec|[|name|] if it [=map/exists=] or |default value| otherwise.

  1. [=map/Set=] |partition key|[|name|] to |value|.

1. For each |name| in the remote end's [=required partition key attributes=]:

  1. If |partition spec|[|name|] [=map/exists=]:

    1. [=map/Set=] |partition key|][|name|] to |partition spec|[|name|].

  1. Otherwise:

    1. Return [=error=] with [=error code=] [=underspecified storage partition=].

1. Return [=success=] with data |partition key|.

</div>

<div algorithm>
To <dfn>get the cookie store</dfn> given |storage partition key|:

1. If |storage partition key| uniquely identifies an extant [=storage partition=]:

  1. Let |store| be the [=cookie store=] of that [=storage partition=].

  1. Return [=success=] with data |store|.

1. Return [=error=] with [=error code=] [=no such storage partition=].

</div>

<div algorithm>
To <dfn>match cookie</dfn> given |stored cookie| and |filter|:

1. For each |name| → |value| in |filter|:

  1. If |stored cookie|[|name|] does not equal |value|:

    1. Return false.

1. Return true.

</div>

<div algorithm>
To <dfn>get matching cookies</dfn> given |cookie store| and |filter|:

1. Let |cookies| be a new list.

1. Set |deserialized filter| to [=deserialize filter=] with |filter|.

1. For each |stored cookie| in |cookie store|:

  1. If [=match cookie=] with |stored cookie| and |deserialized filter| is true:

    1. Append |stored cookie| to |cookies|.

1. Return |cookies|.

</div>

### Commands ### {#module-storage-commands}

#### The storage.getCookies Command ####  {#command-storage-getCookies}

The <dfn export for=commands>storage.getCookies</dfn> command retrieves zero or more [=cookies=] which [=match cookie|match=] a set of provided parameters.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
        storage.GetCookies = (
          method: "storage.getCookies",
          params: storage.GetCookiesParameters
        )

        <!--
        Modifications to this definition should be reflected in
        `network.Cookie`, `network.SetCookieHeader`, and `storage.PartialCookie`.
        -->
        storage.CookieFilter = {
          ? name: text,
          ? value: network.BytesValue,
          ? domain: text,
          ? path: text,
          ? size: js-uint,
          ? httpOnly: bool,
          ? secure: bool,
          ? sameSite: network.SameSite,
          ? expiry: js-uint,
          Extensible,
        }

        storage.BrowsingContextPartitionDescriptor = {
          type: "context",
          context: browsingContext.BrowsingContext
        }

        storage.StorageKeyPartitionDescriptor = {
          type: "storageKey",
          ? userContext: text,
          ? sourceOrigin: text,
          Extensible,
        }

        storage.PartitionDescriptor = (
          storage.BrowsingContextPartitionDescriptor /
          storage.StorageKeyPartitionDescriptor
        )

        storage.GetCookiesParameters = {
          ? filter: storage.CookieFilter,
          ? partition: storage.PartitionDescriptor,
        }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      storage.GetCookiesResult = {
        cookies: [*network.Cookie],
        partitionKey: storage.PartitionKey,
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for storage.getCookies">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |filter| be the value of the <code>filter</code> field of |command parameters|
  if it is present or an empty [=/map=] if it isn't.

1. Let |partition spec| be the value of the <code>partition</code> field of
  |command parameters| if it is present or null if it isn't.

1. Let |partition key| be the result of [=trying=] to [=expand a storage partition spec=] with |partition spec|.

1. Let |store| be the result of [=trying=] to [=get the cookie store=] with
  |partition key|.

1. Let |cookies| be the result of [=get matching cookies=] with |store| and |filter|.

1. Let |serialized cookies| be a new list.

1. For each |cookie| in |cookies|:

  1. Let |serialized cookie| be the result of [=serialize cookie=] given |cookie|.

  1. Append |serialized cookie| to |serialized cookies|.

1. Let |body| be a [=/map=] matching the <code>storage.GetCookiesResult</code> production,
  with the <code>cookies</code> field set to |serialized cookies| and the <code>partitionKey</code>
  field set to |partition key|.

1. Return [=success=] with data |body|.

</div>

#### The storage.setCookie Command ####  {#command-storage-setCookie}

The <dfn export for=commands>storage.setCookie</dfn> command creates a new [=cookie=] in a cookie store, replacing any cookie in that store which matches according to [[COOKIES]].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
        storage.SetCookie = (
          method: "storage.setCookie",
          params: storage.SetCookieParameters,
        )

        <!--
        Modifications to this definition should be reflected in
        `network.Cookie`, `network.SetCookieHeader`, and `storage.CookieFilter`.
        -->
        storage.PartialCookie = {
          name: text,
          value: network.BytesValue,
          domain: text,
          ? path: text,
          ? httpOnly: bool,
          ? secure: bool,
          ? sameSite: network.SameSite,
          ? expiry: js-uint,
          Extensible,
        }

        storage.SetCookieParameters = {
          cookie: storage.PartialCookie,
          ? partition: storage.PartitionDescriptor,
        }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="local-cddl">
      storage.SetCookieResult = {
        partitionKey: storage.PartitionKey
      }
    </pre>
   </dd>
</dl>

<div algorithm="remote end steps for storage.setCookie">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |cookie spec| be the value of the <code>cookie</code> field of |command parameters|.

1. Let |partition spec| be the value of the <code>partition</code> field of
  |command parameters| if it is present or null if it isn't.

1. Let |partition key| be the result of [=trying=] to [=expand a storage partition spec=] with |partition spec|.

1. Let |store| be the result of [=trying=] to [=get the cookie store=] with
  |partition key|.

1. Let |deserialized value| be [=deserialize protocol bytes=] with
  |cookie spec|["<code>value</code>"].

1. [=Create a cookie=] in |store| using [=cookie name=] |cookie spec|["<code>name</code>"],
  [=cookie value=] |deserialized value|, [=cookie domain=]
  |cookie spec|["<code>domain</code>"], and an attribute-value list of the
  following cookie concepts listed in the [=table for cookie conversion=]:

  <dl>
   <dt>[=Cookie path=]
   <dd><p>|cookie spec|["<code>path</code>"] if it exists, otherwise "<code>/</code>".

   <dt>[=Cookie secure only=]
   <dd><p>|cookie spec|["<code>secure</code>"] if it exists, otherwise false.

   <dt>[=Cookie HTTP only=]
   <dd><p>|cookie spec|["<code>httpOnly</code>"] if it exists, otherwise false.

   <dt>[=Cookie expiry time=]
   <dd><p>|cookie spec|["<code>expiry</code>"] if it exists, otherwise leave unset to
    indicate that this is a session cookie.

    Note: The cookie's expiry value might be limited by the remote end in accordance with the [=Cookie Lifetime Limits=].

   <dt>[=Cookie same site=]
   <dd><p>|cookie spec|["<code>sameSite</code>"] if it exists, otherwise leave unset to
    indicate that no same site policy is defined.
  </dl>

  If this step is aborted without inserting a cookie into the cookie store, return
  [=error=] with [=error code=] [=unable to set cookie=].

1. Let |body| be a [=/map=] matching the <code>storage.SetCookieResult</code> production,
  with the <code>partitionKey</code> field set to |partition key|.

1. Return [=success=] with data |body|.

</div>

#### The storage.deleteCookies Command ####  {#command-storage-deleteCookies}

The <dfn export for=commands>storage.deleteCookies</dfn> command removes zero or more [=cookies=] which [=match cookie|match=] a set of provided parameters.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
        storage.DeleteCookies = (
          method: "storage.deleteCookies",
          params: storage.DeleteCookiesParameters,
        )

        storage.DeleteCookiesParameters = {
          ? filter: storage.CookieFilter,
          ? partition: storage.PartitionDescriptor,
        }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
     <pre class="cddl" data-cddl-module="local-cddl">
        storage.DeleteCookiesResult = {
          partitionKey: storage.PartitionKey
        }
     </pre>
   </dd>
</dl>

<div algorithm="remote end steps for storage.deleteCookies">
The [=remote end steps=] with <var ignore>session</var> and |command parameters| are:

1. Let |filter| be the value of the <code>filter</code> field of |command parameters|
  if it is present or an empty [=/map=] if it isn't.

1. Let |partition spec| be the value of the <code>partition</code> field of
  |command parameters| if it is present or null if it isn't.

1. Let |partition key| be the result of [=trying=] to [=expand a storage partition spec=] with |partition spec|.

1. Let |store| be the result of [=trying=] to [=get the cookie store=] with
  |partition key|.

1. Let |cookies| be the result of [=get matching cookies=] with |store| and |filter|.

1. For each |cookie| in |cookies|:

  1. Remove |cookie| from |store|.

1. Let |body| be a [=/map=] matching the <code>storage.DeleteCookiesResult</code> production,
  with the <code>partitionKey</code> field set to |partition key|.

1. Return [=success=] with data |body|.

</div>

## The log Module ## {#module-log}

The <dfn export for=modules>log</dfn> module contains functionality and events
related to logging.

A [=BiDi Session=] has a <dfn>log event buffer</dfn> which is a [=/map=] from
[=navigable id=] to a list of log events for that context that have not
been emitted. User agents may impose a maximum size on this buffer, subject to
the condition that if events A and B happen in the same context with A occurring
before B, and both are added to the buffer, the entry for B must not be removed
before the entry for A.

<div algorithm>

To <dfn>buffer a log event</dfn> given |session|, |navigables| and |event|:

1. Let |buffer| be |session|'s [=log event buffer=].

1. Let |navigable ids| be a new list.

1. For each |navigable| of |navigables|:

  1. Append the [=navigable id=] for |navigable| to |navigable ids|.

1. For each |navigable id| in |navigable ids|:

  1. Let |other navigables| be an empty [=/list=]

  1. For each |other id| in |navigable ids|:

   1. If |other id| is not equal to |navigable id|, append |other id| to |other
      navigables|.

  1. If |buffer| does not contain |navigable id|, let |buffer|[|navigable id|] be a
     new list.

  1. Append (|event|, |other navigables|) to |buffer|[|navigable id|].

Note: we store the other navigables here so that each event is only emitted
once. In practice this is only relevant for workers that can be associated with
multiple navigables.

Issue: Do we want to key this on browsing context or top-level traversable?
The difference is in what happens if an event occurs in a frame and that frame
is then navigated before the local end subscribes to log events for the top
level navigable.

</div>

### Definition ### {#module-log-definition}

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
LogEvent = (
  log.EntryAdded
)
</pre>

### Types ### {#module-log-types}

#### log.LogEntry #### {#types-log-logentry}

{^Local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
log.Level = "debug" / "info" / "warn" / "error"

log.Entry = (
  log.GenericLogEntry /
  log.ConsoleLogEntry /
  log.JavascriptLogEntry
)

log.BaseLogEntry = (
  level: log.Level,
  source: script.Source,
  text: text / null,
  timestamp: js-uint,
  ? stackTrace: script.StackTrace,
)

log.GenericLogEntry = {
  log.BaseLogEntry,
  type: text,
}

log.ConsoleLogEntry = {
  log.BaseLogEntry,
  type: "console",
  method: text,
  args: [*script.RemoteValue],
}

log.JavascriptLogEntry = {
  log.BaseLogEntry,
  type: "javascript",
}
</pre>

Each log event is represented by a <code>log.Entry</code> object. This has a
<code>type</code> property which represents the type of log entry added, a
<code>level</code> property representing severity, a <code>source</code>
property representing the origin of the log entry, a <code>text</code> property
with the log message string itself, and a <code>timestamp</code> property
corresponding to the time the log entry was generated. Specific variants of the
<code>log.Entry</code> are used to represent logs from different sources, and
provide additional fields specific to the entry type.

### Events ### {#module-log-events}

#### The log.entryAdded Event #### {#event-log-entryAdded}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
        log.EntryAdded = (
         method: "log.entryAdded",
         params: log.Entry,
        )
      </pre>
   </dd>
</dl>

The [=remote end event trigger=] is:

<div algorithm="remote end event trigger for log.entryAdded">

Define the following [=console steps=] with |method|, |args|, and
<var ignore>options</var>:

1. For each |session| in [=active BiDi sessions=]:

  1. If |method| is "<code>error</code>" or "<code>assert</code>", let |level| be
     "<code>error</code>". If |method| is "<code>debug</code>" or "<code>trace</code>"
     let |level| be "<code>debug</code>". If |method| is "<code>warn</code>", let
     |level| be "<code>warn</code>". Otherwise let |level| be "<code>info</code>".

  1. Let |timestamp| be a [=time value=] representing the current date and time in UTC.

  1. Let |text| be an empty string.

  1. If [=/Type=](|args|[0]) is String, and |args|[0] contains a
     [=formatting specifier=], let |formatted args| be [=Formatter=](|args|). Otherwise
     let |formatted args| be |args|.

     Note: The formatter operation is underdefined in the console specification,
     formatting can be inconsistent between different implementations.

  1. For each |arg| in |formatted args|:

    1. If |arg| is not the first entry in |args|, append a U+0020 SPACE to |text|.

    1. If |arg| is a [=primitive ECMAScript value=], append [=ToString=](|arg|) to
       |text|. Otherwise append an implementation-defined string to |text|.

  1. Let |realm| be the [=realm id=] of the [=current Realm Record=].

  1. Let |serialized args| be a new list.

  1. Let |serialization options| be a [=/map=] matching the
     <code>script.SerializationOptions</code> production with the fields set to
     their default values.

  1. For each |arg| of |args|:

     1. Let |serialized arg| be the result of [=serialize as a remote value=] with
        |arg| as value, |serialization options|, <code>none</code> as
        ownership type, a new [=/map=] as serialization internal map, |realm| and
        |session|.

     1. Add |serialized arg| to |serialized args|.

  1. Let |source| be the result of [=get the source=] given [=current Realm Record=].

  1. Let |stack| be the [=current stack trace=].

  1. Let |entry| be a [=/map=] matching the <code>log.ConsoleLogEntry</code> production,
     with the the <code>level</code> field set to |level|, the <code>text</code>
     field set to |text|, the <code>timestamp</code> field set to |timestamp|, the
     <code>stackTrace</code> field set to |stack|, the |method| field set to |method|,
     the <code>source</code> field set to |source|,
     and the <code>args</code> field set to |serialized args|.

  1. Let |body| be a [=/map=] matching the <code>log.EntryAdded</code> production, with
     the <code>params</code> field set to |entry|.

  1. Let |settings| be the [=current settings object=]

  1. Let |related navigables| be the result of [=get related navigables=] given |settings|.

  1. If [=event is enabled=] with |session|, "<code>log.entryAdded</code>" and
     |related navigables|, [=emit an event=] with |session| and |body|.

     Otherwise, [=buffer a log event=] with |session|, |related browsing
     contexts|, and |body|.

Define the following [=error reporting steps=] with arguments |script|, <var
ignore>line number</var>, <var ignore>column number</var>, |message| and
|handled|:

1. If |handled| is true return.

1. Let |settings| be |script|'s [=script/settings object=].

1. Let |timestamp| be a [=time value=] representing the current date and time in UTC.

1. Let |stack| be the [=stack trace for an exception=] with the exception
   corresponding to the error being reported.

1. Let |source| be the result of [=get the source=] given [=current Realm Record=].

1. Let |entry| be a [=/map=] matching the <code>log.JavascriptLogEntry</code> production,
   with <code>level</code> set to "<code>error</code>", <code>text</code> set to
   |message|, <code>source</code> set to |source|, <code>timestamp</code>
   set to |timestamp|, and the <code>stackTrace</code> field set to |stack|.

1. Let |body| be a [=/map=] matching the <code>log.EntryAdded</code> production, with
     the <code>params</code> field set to |entry|.

1. Let |related navigables| be the result of [=get related navigables=] given |settings|.

1. For each |session| in [=active BiDi sessions=]:

  1. If [=event is enabled=] with |session|, "<code>log.entryAdded</code>" and
     |related navigables|, [=emit an event=] with |session| and |body|.

     Otherwise, [=buffer a log event=] with |session|, |related browsing
     contexts|, and |body|.

Issue: Lots more things require logging. CDP has LogEntryAdded types xml,
javascript, network, storage, appcache, rendering, security, deprecation,
worker, violation, intervention, recommendation, other. These are in addition to
the js exception and console API types that are represented by different methods.

Issue: Allow implementation-defined log types

</div>

<div algorithm="remote end subscribe steps for log.entryAdded">

The [=remote end subscribe steps=], with [=subscribe priority=] 10, given
|session|, |navigables| and |include global| are:

1. For each |navigable id| → |events| in |session|'s [=log event buffer=]:

  1. Let |maybe context| be the result of [=getting a navigable=] given
     |navigable id|.

  1. If |maybe context| is an [=error=], remove |navigable id| from [=log event
     buffer=] and continue.

  1. Let |navigable| be |maybe context|'s data

  1. Let |top level navigable| be |navigable|'s [=navigable/top-level traversable=].

  1. If |include global| is true and |top level navigable| is not in |navigables|,
     or if |include global| is false and |top level navigable| is in |navigables|:

    1. For each (|event|, |other navigables|) in |events|:

      1. [=Emit an event=] with |session| and |event|.

      1. For each |other context id| in |other navigables|:

        1. If [=log event buffer=] contains |other context id|, remove |event|
           from [=log event buffer=][|other context id|].

</div>

## The input Module ## {#module-input}

The <dfn export for=modules>input</dfn> module contains functionality for
simulated user input.

### Definition ### {#module-input-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
InputCommand = (
  input.PerformActions //
  input.ReleaseActions //
  input.SetFiles
)
</pre>

<pre class="cddl" data-cddl-module="local-cddl">
InputResult = (
  input.PerformActionsResult /
  input.ReleaseActionsResult /
  input.SetFilesResult
)
</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">

InputEvent = (
  input.FileDialogOpened
)
</pre>

### Types ### {#module-input-types}

#### input.ElementOrigin #### {#type-input-origin}

The <code>input.ElementOrigin</code> type represents an {{Element}} that will
be used as a coordinate origin.

<pre class="cddl" data-cddl-module="remote-cddl">
input.ElementOrigin = {
  type: "element",
  element: script.SharedReference
}
</pre>

<div algorithm>
The <dfn>is <code>input.ElementOrigin</code></dfn> steps given |object| are:

1. If |object| is a [=/map=] matching the <code>input.ElementOrigin</code>
   production, return true.

1. Return false.

</div>

<div algorithm>To <dfn>get Element from <code>input.ElementOrigin</code>
steps</dfn> given |session|:

1. Return the following steps, given |origin| and |navigable|:

  1. Assert: |origin| matches <code>input.ElementOrigin</code>.

  <!-- converting to realm here is kind of silly since we immediately convert back
       in [=deserialize remote reference=] -->
  1. Let |document| be |navigable|'s [=active document=].

  1. Let |reference| be |origin|["<code>element</code>"]

  1. Let |environment settings| be the [=environment settings object=] whose
     [=relevant global object=]'s <a>associated <code>Document</code></a> is
     |document|.

  1. Let |realm| be |environment settings|' [=realm execution context=]'s
     Realm component.

  1. Let |element| be the result of [=trying=] to [=deserialize remote reference=]
     with |reference|, |realm|, and |session|.

  1. If |element| doesn't implement {{Element}} return [=error=] with [=error code=]
     [=no such element=].

  1. Return [=success=] with data |element|.

</div>

### Commands ### {#module-input-commands}

#### The input.performActions Command ####  {#command-input-performActions}

The <dfn export for=commands>input.performActions</dfn> command performs a
specified sequence of user input actions.

Note: for a detailed description of the behavior of this command, see the
[=actions=] section of [[WEBDRIVER]].

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      input.PerformActions = (
        method: "input.performActions",
        params: input.PerformActionsParameters
      )

      input.PerformActionsParameters = {
        context: browsingContext.BrowsingContext,
        actions: [*input.SourceActions]
      }

      input.SourceActions = (
        input.NoneSourceActions /
        input.KeySourceActions /
        input.PointerSourceActions /
        input.WheelSourceActions
      )

      input.NoneSourceActions = {
        type: "none",
        id: text,
        actions: [*input.NoneSourceAction]
      }

      input.NoneSourceAction = input.PauseAction

      input.KeySourceActions = {
        type: "key",
        id: text,
        actions: [*input.KeySourceAction]
      }

      input.KeySourceAction = (
        input.PauseAction /
        input.KeyDownAction /
        input.KeyUpAction
      )

      input.PointerSourceActions = {
        type: "pointer",
        id: text,
        ? parameters: input.PointerParameters,
        actions: [*input.PointerSourceAction]
      }

      input.PointerType = "mouse" / "pen" / "touch"

      input.PointerParameters = {
        ? pointerType: input.PointerType .default "mouse"
      }

      input.PointerSourceAction = (
        input.PauseAction /
        input.PointerDownAction /
        input.PointerUpAction /
        input.PointerMoveAction
      )

      input.WheelSourceActions = {
        type: "wheel",
        id: text,
        actions: [*input.WheelSourceAction]
      }

      input.WheelSourceAction = (
        input.PauseAction /
        input.WheelScrollAction
      )

      input.PauseAction = {
        type: "pause",
        ? duration: js-uint
      }

      input.KeyDownAction = {
        type: "keyDown",
        value: text
      }

      input.KeyUpAction = {
        type: "keyUp",
        value: text
      }

      input.PointerUpAction = {
        type: "pointerUp",
        button: js-uint,
      }

      input.PointerDownAction = {
        type: "pointerDown",
        button: js-uint,
        input.PointerCommonProperties
      }

      input.PointerMoveAction = {
        type: "pointerMove",
        x: float,
        y: float,
        ? duration: js-uint,
        ? origin: input.Origin,
        input.PointerCommonProperties
      }

      input.WheelScrollAction = {
        type: "scroll",
        x: js-int,
        y: js-int,
        deltaX: js-int,
        deltaY: js-int,
        ? duration: js-uint,
        ? origin: input.Origin .default "viewport",
      }

      input.PointerCommonProperties = (
        ? width: js-uint,
        ? height: js-uint,
        ? pressure: (0.0..1.0),
        ? tangentialPressure: (-1.0..1.0),
        ? twist: (0..359),
        ; 0 .. Math.PI / 2
        ? altitudeAngle: (0.0..1.5707963267948966),
        ; 0 .. 2 * Math.PI
        ? azimuthAngle: (0.0..6.283185307179586),
      )

      input.Origin = "viewport" / "pointer" / input.ElementOrigin
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      input.PerformActionsResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for input.performActions">

The [=remote end steps=] with |session| and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Let |input state| be [=get the input state=] with |session| and |navigable|'s
   [=navigable/top-level traversable=].

1. Let |actions options| be a new [=actions options=] with the [=is element
   origin=] steps set to [=is input.ElementOrigin=], and the
   [=get element origin=] steps set to the result of [=get Element from
   input.ElementOrigin steps=] given |session|.

1. Let |actions by tick| be the result of trying to [=extract an action
   sequence=] with |input state|, |command parameters|, and |actions options|.

1. [=Try=] to [=dispatch actions=] with |input state|, |actions by tick|,
   |navigable|, and |actions options|.

1. Return [=success=] with data null.

</div>

#### The input.releaseActions Command ####  {#command-input-releaseActions}

The <dfn export for=commands>input.releaseActions</dfn> command resets the input
state associated with the current session.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      input.ReleaseActions = (
        method: "input.releaseActions",
        params: input.ReleaseActionsParameters
      )

      input.ReleaseActionsParameters = {
        context: browsingContext.BrowsingContext,
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      input.ReleaseActionsResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for input.releaseActions">

The [=remote end steps=] given |session|, and |command parameters| are:

1. Let |navigable id| be the value of the <code>context</code> field of
   |command parameters|.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=]
   with |navigable id|.

1. Let |top-level traversable| be |navigable|'s [=navigable/top-level traversable=].

1. Let |input state| be [=get the input state=] with |session| and |top-level traversable|.

1. Let |actions options| be a new [=actions options=] with the [=is element
   origin=] steps set to [=is input.ElementOrigin=], and the
   [=get element origin=] steps set to [=get Element from
   input.ElementOrigin steps=] given |session|.

1. Let |undo actions| be |input state|’s [=input cancel list=] in reverse order.

1. [=Try=] to [=dispatch tick actions=] with |undo actions|, 0, |navigable|, and
   |actions options|.

1. [=Reset the input state=] with |session| and |top-level traversable|.

1. Return [=success=] with data null.

</div>

#### The input.setFiles Command ####  {#command-input-setFiles}

The <dfn export for=commands>input.setFiles</dfn> command sets the <code>files</code> property of a given <code>input</code> element with type <code>file</code>
to a set of file paths.

<dl>
   <dt>Command Type</dt>
   <dd>
    <pre class="cddl" data-cddl-module="remote-cddl">
      input.SetFiles = (
        method: "input.setFiles",
        params: input.SetFilesParameters
      )

      input.SetFilesParameters = {
        context: browsingContext.BrowsingContext,
        element: script.SharedReference,
        files: [*text]
      }
    </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      input.SetFilesResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for input.setFiles">

The [=remote end steps=] given |session| and |command parameters| are:

1. Let |navigable id| be the value of the |command
   parameters|["<code>context</code>"] field.

1. Let |navigable| be the result of [=trying=] to [=get a navigable=] with
   |navigable id|.

1. Let |document| be |navigable|'s [=active document=].

1. Let |environment settings| be the [=environment settings object=] whose
   [=relevant global object=]'s <a>associated <code>Document</code></a> is
   |document|.

1. Let |realm| be |environment settings|'s [=realm execution context=]'s
   Realm component.

1. Let |element| be the result of [=trying=] to [=deserialize remote reference=]
   with |command parameters|["<code>element</code>"], |realm|, and |session|.

1. If |element| doesn't implement {{Element}}, return [=error=] with [=error code=]
   [=no such element=].

1. If |element| doesn't implement {{HTMLInputElement}}, |element|'s
   <{input/type}> is not in the [=File Upload state=], or |element| is [=disabled=],
   return [=error=] with [=error code=] [=unable to set file input=].

1. If the [=set/size=] of |files| is greater than 1 and |element|'s
   <{input/multiple}> attribute is not set, return [=error=] with [=error code=]
   [=unable to set file input=].

1. Let |files| be the value of the |command parameters|["<code>files</code>"]
   field.

1. Let |selected files| be |element|'s [=selected files=].

1. If the [=set/size=] of the [=set/intersection=] of |files| and |selected
   files| is equal to the [=set/size=] of |selected files| and equal to the
   [=set/size=] of |files|, [=queue an element task=] on the [=user interaction task source=]
   given |element| to fire an event named <code>cancel</code> at |element|, with
   the <code>bubbles</code> attribute initialized to true.

   Note: Cancellation in a browser is typically determined by changes in file
   selection. In other words, if there is no change, a "cancel" event is sent.

1. Otherwise, [=update the file selection=] for |element| with |files| as the
   user's selection.

1. If, for any reason, the remote end is unable to set the [=selected files=] of |element| to the files with paths given in |files|,
   return error with [=error code=] [=unsupported operation=].

   Note: For example remote ends might be unable to set [=selected files=] to files that do not currently exist on the filesystem.

1. Return [=success=] with data null.

</div>

### Events ### {#module-input-events}

#### The input.fileDialogOpened Event #### {#event-input-fileDialogOpened}

<dl>
   <dt>Event Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
         input.FileDialogOpened = (
            method: "input.fileDialogOpened",
            params: input.FileDialogInfo
         )

         input.FileDialogInfo = {
            context: browsingContext.BrowsingContext,
            ? userContext: browser.UserContext,
            ? element: script.SharedReference,
            multiple: bool,
         }
      </pre>
   </dd>
</dl>

A <dfn>WebDriver BiDi file picker options</dfn> is a [=struct=] with an [=struct/item=] named
<dfn for="WebDriver BiDi file picker options">multiple</dfn> which is a boolean.

<div algorithm>
The [=remote end event trigger=] is the
<dfn export>WebDriver BiDi file dialog opened</dfn> steps, given [=/element=] |element| and
optionally [=WebDriver BiDi file picker options=] |file picker options| (default: null):

Note: unlike other user prompt handlers, the default behavior is to allow for the file dialog to be
opened.

1. Let |navigable| be the |element|'s [=node document=]'s [=/navigable=].

1. Let |navigable id| be |navigable|'s [=navigable id=].

1. Let |user context id| be the [=user context id=] of |navigable|'s [=associated user context=].

1. Let |multiple| be <code>false</code>.

1. If |element| is not null and |element|'s <{input/multiple}> attribute is set,
    set |multiple| to <code>true</code>.

1. If |file picker options| is not null and |file picker options|'s [=WebDriver BiDi file picker options/multiple=] is true, set |multiple| to
      <code>true</code>.

1. Let |related navigables| be a [=/set=] containing |navigable|.

1. For each |session| in the [=set of sessions for which an event is enabled=] given
   "<code>input.fileDialogOpened</code>" and |related navigables|:

  1. Let |params| be a [=/map=] matching the <code>input.FileDialogInfo</code>
     production with the <code>context</code> field set to |navigable id|, the
     <code>userContext</code> field set to |user context id| and
     <code>multiple</code> field set to |multiple|.

  1. If |element| is not null:

    1. Let |shared id| be [=get shared id for a node=] with |element| and |session|.

    1. Set |params|["<code>element</code>"] to |shared id|.

  1. Let |body| be a [=/map=] matching the <code>input.fileDialogOpened</code>
     production, with the <code>params</code> field set to |params|.

  1. [=Emit an event=] with |session| and |body|.

1. Let |dismissed| be false.

1. For each |session| in [=active BiDi sessions=]:

  1. Let |user prompt handler| be |session|'s [=user prompt handler=].

  1. If |user prompt handler| is not null:

   1. Assert |user prompt handler| is a [=/map=].

   1. If |user prompt handler| [=map/contains=] "<code>file</code>":

      1. If |user prompt handler|["<code>file</code>"] is not equal to
         "<code>ignore</code>", set |dismissed| to true.

   1. Otherwise if |user prompt handler| [=map/contains=] "<code>default</code>" and
      |user prompt handler|["<code>default</code>"] is not equal to
      "<code>ignore</code>", set |dismissed| to true.

1. Return |dismissed|.

</div>


## The webExtension Module ## {#module-webExtension}

The <dfn export for=modules>webExtension</dfn> module contains functionality for
managing and interacting with web extensions.

### Definition ### {#module-webExtension-definition}

{^remote end definition^}

<pre class="cddl" data-cddl-module="remote-cddl">
WebExtensionCommand = (
  webExtension.Install //
  webExtension.Uninstall
)
</pre>

{^local end definition^}

<pre class="cddl" data-cddl-module="local-cddl">
WebExtensionResult = (
  webExtension.InstallResult /
  webExtension.UninstallResult
)
</pre>

### Types ### {#module-webExtension-types}

#### The webExtension.Extension Type #### {#type-webExtension-Extension}

<pre class="cddl" data-cddl-module="remote-cddl,local-cddl">
webExtension.Extension = text
</pre>

The <code>webExtension.Extension</code> type represents a web extension id within a [=remote end=].

### Commands ### {#module-webExtension-commands}

#### The webExtension.install Command #### {#command-webExtension-install}

The <dfn export for=commands>webExtension.install</dfn> command installs a web extension in the [=remote end=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      webExtension.Install = (
         method: "webExtension.install",
         params: webExtension.InstallParameters
      )

      webExtension.InstallParameters = {
         extensionData: webExtension.ExtensionData,
      }

      webExtension.ExtensionData = (
         webExtension.ExtensionArchivePath /
         webExtension.ExtensionBase64Encoded /
         webExtension.ExtensionPath
      )

      webExtension.ExtensionPath = {
         type: "path",
         path: text,
      }

      webExtension.ExtensionArchivePath = {
         type: "archivePath",
         path: text,
      }

      webExtension.ExtensionBase64Encoded = {
         type: "base64",
         value: text,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      webExtension.InstallResult = {
        extension: webExtension.Extension
      }
      </pre>
   </dd>
</dl>

<div algorithm>
To <dfn>extract a zip archive</dfn> given |bytes|:

1. Perform implementation defined steps to decode |bytes| using the zip compression algorithm. TODO: Find a better reference for zip decoding.

1. If the previous step failed (e.g. because |bytes| did not represent valid zip-compressed data) then return [=error=] with error code [=invalid web extension=]. Otherwise let |entry| be a [=directory entry=] containing the extracted filesystem entries.

1. Return |entry|.

</div>

<div algorithm>
To <dfn>expand a web extension data spec</dfn> given |extension data spec|:

1. Let |type| be |extension data spec|["<code>type</code>"].

1. If installing a web extension using |type| isn't supported return [=error=] with [=error code=] [=unsupported operation=].

1. In the following list of conditions and associated steps, run the first set of steps for which the associated condition is true:

   <dl>
    <dt>|type| is the string "<code>path</code>"
    <dd>
      1. Let |path| be |extension data spec|["<code>path</code>"].
      1. Let |locator| be a [=directory locator=] with [=file system locator/path=] |path| and [=file system locator/root=] corresponding to the root of the file system.
      1. Let |entry| be [=locate an entry=] given |locator|.

    <dt>|type| is the string "<code>archivePath</code>"
    <dd>
      1. Let |archive path| be |extension data spec|["<code>path</code>"].
      1. Let |locator| be a [=file locator=] with [=file system locator/path=] |archive path| and [=file system locator/root=] corresponding to the root of the file system.
      1. Let |archive entry| be [=locate an entry=] given |locator|.
      1. If |archive entry| is null, return null.
      1. Let |bytes| be |archive entry|'s [=file entry/binary data=].
      1. Let |entry| be the result of [=trying=] to [=extract a zip archive=] given |bytes|.

    <dt>|type| is the string "<code>base64</code>"
    <dd>
      1. Let |bytes| be [=forgiving-base64 decode=] |extension data spec|["<code>value</code>"].
      1. If |bytes| is failure, return null.
      1. Let |entry| be the result of [=trying=] to [=extract a zip archive=] given |bytes|.

   </dl>

1. Return |entry|.

</div>

<div algorithm="remote end steps for webExtension.install">
The [=remote end steps=] with |command parameters| are:

1. If installing web extensions isn't supported return [=error=] with error code [=unsupported operation=].

1. Let |extension data spec| be |command parameters|["<code>extensionData</code>"].

1. Let |extension directory entry| be the result of [=trying=] to [=expand a web extension data spec=] with |extension data spec|.

1. If |extension directory entry| is null, return [=error=] with [=error code=] [=invalid web extension=].

1. Perform implementation defined steps to install a web extension from |extension directory entry|. If this fails, return [=error=] with [=error code=] [=invalid web extension=]. Otherwise let |extension id| be the unique identifier of the newly installed web extension.

1. Let |result| be a [=/map=] matching the
   <code>webExtension.InstallResult</code> production with the
   <code>extension</code> field set to |extension id|.

1. Return [=success=] with data |result|.

</div>

Note: Browsers might install the web extension only temporarily by default so
that they will be automatically uninstalled during the next shutdown.

#### The webExtension.uninstall Command #### {#command-webExtension-uninstall}

The <dfn export for=commands>webExtension.uninstall</dfn> command uninstalls a web extension for the [=remote end=].

<dl>
   <dt>Command Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="remote-cddl">
      webExtension.Uninstall = (
         method: "webExtension.uninstall",
         params: webExtension.UninstallParameters
      )

      webExtension.UninstallParameters = {
         extension: webExtension.Extension,
      }
      </pre>
   </dd>
   <dt>Return Type</dt>
   <dd>
      <pre class="cddl" data-cddl-module="local-cddl">
      webExtension.UninstallResult = EmptyResult
      </pre>
   </dd>
</dl>

<div algorithm="remote end steps for webExtension.uninstall">
The [=remote end steps=] with |command parameters| are:

1. Let |extension| be |command parameters|["<code>extension</code>"].

1. If the [=remote end=] has no web extension with id equal to |extension|, return [=error=] with [=error code=] [=no such web extension=].

1. Perform any implementation-defined steps to remove the web extension from the [=remote end=]. If this fails, return [=error=] with [=error code=] [=unknown error=].

1. Return [=success=] with data null.

</div>

# Patches to Other Specifications # {#patches}

This specification requires some changes to external specifications to provide the necessary
integration points. It is assumed that these patches will be committed to the other specifications
as part of the standards process.

## HTML ##  {#patches-html}

The [=report an error=] algorithm is modified with an additional step at the
end:

<div algorithm>
1. Call any <dfn>error reporting steps</dfn> defined in external specifications
   with <var ignore>script</var>, <var ignore>line</var>, <var
   ignore>col</var>, <var ignore>message</var>, and true if the error is
   handled, or false otherwise.

</div>
## Console ##  {#patches-console}

Other specifications can define <dfn>console steps</dfn>.

1. At the point when the [=Printer=] operation is called with
   arguments |name|, |printerArgs| and |options| (which is undefined if the
   argument is not provided), call any [=console steps=] defined in
   external specification with arguments |name|, |printerArgs|, and |options|.


## CSS ## {#patchs-css}

### Determine the device pixel ratio ### {#patchs-determine-the-device-pixel-ratio}

Insert the following steps at the start of the [=determine the device pixel ratio=] algorithm:

1. If [=device pixel ratio overrides=] [=map/contains=] <var ignore>window</var>'s [=window/navigable=], return [=device pixel ratio overrides=][<var ignore>window</var>'s [=window/navigable=]].

### Evaluating Media Queries ### {#patchs-evaluating-media-queries}

Issue: Remove after https://github.com/w3c/csswg-drafts/pull/13549 is merged.

The [<dfn export>evaluate media feature</dfn> algorithm is modified to support WebDriver BiDi
overrides. Replace the step "The result is the result of evaluating the specified media feature" to:

<div algorithm="evaluate media feature">
To get the result of evaluating a [=media feature=] |media feature| in the given [=/Document=] |document|:

1. Let |media feature name| be the |media feature|'s [=mf-name=].

1. Let |emulated value| be the result of getting [=WebDriver BiDi media feature value=] for the
   |document| and |media feature name|.

1. If |emulated value| is not null, return |emulated value|.

1. Otherwise, return the result of evaluating the specified |media feature|.

</div>

### The viewport meta element ### {#patches-viewport-meta-element}

Issue: remove after https://github.com/w3c/csswg-drafts/pull/13548 is merged.

The 'Viewport meta element' section of the [[!CSS-VIEWPORT-1]] specification is modified to verify the
[=WebDriver BiDi viewport meta state=].

If [=WebDriver BiDi viewport meta state=] given the `viewport` meta element's
[=node document=] is true, the user agent MUST use the `viewport` meta element.

Otherwise, the user agent MAY use the `viewport` meta element.

# Appendices # {#appendices}

<em>This section is non-normative.</em>

## External specifications ## {#external-specifications}

Note: the list is not exhaustive and might not be up to date.

The following external specifications define additional WebDriver BiDi modules:

1. <a href="https://www.w3.org/TR/digital-credentials/#automated-testing">Digital Credentials API</a>

1. <a href="https://www.w3.org/TR/permissions/#automation-webdriver-bidi">Permissions</a>

1. <a href="https://wicg.github.io/nav-speculation/prefetch.html#automated-testing">nav-speculation</a>

1. <a href="https://wicg.github.io/ua-client-hints/#automation">User-Agent Client Hints</a>

1. <a href="https://webbluetoothcg.github.io/web-bluetooth/#automated-testing">Web Bluetooth</a>
