Component({
  properties: {
    state: { type: String, value: "loading" },
    loadingText: { type: String, value: "" },
    emptyText: { type: String, value: "" },
    emptyActionText: { type: String, value: "" },
    errorText: { type: String, value: "" },
    errorSubText: { type: String, value: "" },
    errorActionText: { type: String, value: "" },
    offlineText: { type: String, value: "" },
    offlineSubText: { type: String, value: "" },
    offlineActionText: { type: String, value: "" }
  },
  methods: {
    onAction() {
      this.triggerEvent("action");
    }
  }
});

