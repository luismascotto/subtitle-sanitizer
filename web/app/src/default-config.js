export const defaultConfig = {
  removeTextBeforeColonIfUppercase: true,
  removeSingleLineColon: true,
  removeBetweenDelimiters: [
    { left: "(", right: ")" },
    { left: "[", right: "]" },
    { left: "{", right: "}" },
    { left: "*", right: "*" },
    { left: "♪", right: "♪" },
  ],
  removeLineIfContains: " music *\n music ♪",
};

export const defaultConfigText = JSON.stringify(defaultConfig, null, 2);
