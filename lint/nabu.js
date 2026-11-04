// nabu's own lint rules: the ones oxlint has no built-in for. oxlint.config.ts loads this file as a JS plugin.

// The props through which a Mantine part takes its color.
const COLOR = new Set(["color", "variant", "c", "bg"]);

export default {
  meta: { name: "nabu" },
  rules: {
    // A ui part writes its color as a string, never picks it by a condition: the meaning of a task (late,
    // closed) stays out of how ui looks.
    "no-conditional-color": {
      meta: {
        type: "problem",
        messages: {
          state: "Write {{name}} as a string instead: a ui part does not pick its color by state",
        },
      },
      create(context) {
        return {
          JSXAttribute(node) {
            const name = node.name.name;
            if (COLOR.has(name) && node.value?.type === "JSXExpressionContainer") {
              context.report({ node, messageId: "state", data: { name } });
            }
          },
        };
      },
    },
  },
};
