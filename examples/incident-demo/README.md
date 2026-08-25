# Agent incident demo

This is ATB's flagship offline workflow. A release agent attempts a production
deployment outside the allowed change window. The ordinary application log
records only a generic failure and request close; it omits the tool name, policy
decision, reviewer context, and evidence order.

Run it from a fresh source checkout:

```bash
make demo-incident
```

The target builds the local CLI, uses the public Python SDK action and oversight
helpers to create `run.atb/incident-demo/incident.atb`, proves the bytes are
deterministic across two generations, verifies the intact chain, lists and
reports the session, asserts the `tool_without_approval` finding, and confirms
that content mutation, record reordering, and record removal each exit with
integrity status 2. It also asserts that the ordinary log does not contain the
tool, denial reason, reviewer, or finding needed to reconstruct the incident.

Review the same evidence manually:

```bash
cat examples/incident-demo/application.log
./atb incident list --bundle run.atb/incident-demo/incident.atb
./atb incident report \
  --bundle run.atb/incident-demo/incident.atb \
  --session sess-incident-demo
./atb view --bundle run.atb/incident-demo/incident.atb
```

The verified bundle proves the presented record order and content have not
changed. `tool_without_approval` means no matching approval appears earlier in
the captured evidence; it does not prove that no approval existed outside the
instrumented boundary. The reviewer identity object is also caller-provided
until its retained assertion is checked independently.
