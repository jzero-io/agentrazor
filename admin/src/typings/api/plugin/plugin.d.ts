declare namespace Api {
  /** backend api module: "plugin" */
  namespace Plugin {
    type AdminLocaleMessages = Record<string, unknown>;

    interface AdminLocalesResponse {
      locales: Partial<Record<App.I18n.LangType, AdminLocaleMessages>>;
    }
  }
}
