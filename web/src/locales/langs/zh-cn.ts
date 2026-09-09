const local: App.I18n.Schema = {
  system: {
    title: 'agentrazor-admin',
    updateTitle: '系统版本更新通知',
    updateContent: '检测到系统有新版本发布，是否立即刷新页面？',
    updateConfirm: '立即刷新',
    updateCancel: '稍后再说'
  },
  common: {
    action: '操作',
    add: '新增',
    addSuccess: '添加成功',
    backToHome: '返回首页',
    batchDelete: '批量删除',
    cancel: '取消',
    close: '关闭',
    check: '勾选',
    expandColumn: '展开列',
    columnSetting: '列设置',
    config: '配置',
    confirm: '确认',
    delete: '删除',
    deleteSuccess: '删除成功',
    confirmDelete: '确认删除吗？',
    edit: '编辑',
    editSuccess: '编辑成功',
    editHomeSuccess: '编辑角色首页成功',
    warning: '警告',
    error: '错误',
    index: '序号',
    keywordSearch: '请输入关键词搜索',
    logout: '退出登录',
    logoutConfirm: '确认退出登录吗？',
    lookForward: '敬请期待',
    modify: '修改',
    modifySuccess: '修改成功',
    noData: '无数据',
    operate: '操作',
    pleaseCheckValue: '请检查输入的值是否合法',
    refresh: '刷新',
    reset: '重置',
    save: '保存',
    search: '搜索',
    switch: '切换',
    tip: '提示',
    trigger: '触发',
    update: '更新',
    updateSuccess: '更新成功',
    userCenter: '个人中心',
    yesOrNo: {
      yes: '是',
      no: '否'
    }
  },
  request: {
    unauthorized: '登录状态已失效，请重新登录',
    forbidden: '没有权限执行此操作',
    logout: '请求失败后登出用户',
    logoutMsg: '用户状态失效，请重新登录',
    logoutWithModal: '请求失败后弹出模态框再登出用户',
    logoutWithModalMsg: '用户状态失效，请重新登录',
    refreshToken: '请求的token已过期，刷新token',
    tokenExpired: 'token已过期'
  },
  theme: {
    themeSchema: {
      title: '主题模式',
      light: '亮色模式',
      dark: '暗黑模式',
      auto: '跟随系统'
    },
    grayscale: '灰色模式',
    colourWeakness: '色弱模式',
    layoutMode: {
      title: '布局模式',
      vertical: '左侧菜单模式',
      'vertical-mix': '左侧菜单混合模式',
      horizontal: '顶部菜单模式',
      'horizontal-mix': '顶部菜单混合模式',
      reverseHorizontalMix: '一级菜单与子级菜单位置反转'
    },
    recommendColor: '应用推荐算法的颜色',
    recommendColorDesc: '推荐颜色的算法参照',
    themeColor: {
      title: '主题颜色',
      primary: '主色',
      info: '信息色',
      success: '成功色',
      warning: '警告色',
      error: '错误色',
      followPrimary: '跟随主色'
    },
    scrollMode: {
      title: '滚动模式',
      wrapper: '外层滚动',
      content: '主体滚动'
    },
    page: {
      animate: '页面切换动画',
      mode: {
        title: '页面切换动画类型',
        'fade-slide': '滑动',
        fade: '淡入淡出',
        'fade-bottom': '底部消退',
        'fade-scale': '缩放消退',
        'zoom-fade': '渐变',
        'zoom-out': '闪现',
        none: '无'
      }
    },
    fixedHeaderAndTab: '固定头部和标签栏',
    header: {
      height: '头部高度',
      breadcrumb: {
        visible: '显示面包屑',
        showIcon: '显示面包屑图标'
      }
    },
    tab: {
      visible: '显示标签栏',
      cache: '缓存标签页',
      height: '标签栏高度',
      mode: {
        title: '标签栏风格',
        chrome: '谷歌风格',
        button: '按钮风格'
      }
    },
    sider: {
      inverted: '深色侧边栏',
      width: '侧边栏宽度',
      collapsedWidth: '侧边栏折叠宽度',
      mixWidth: '混合布局侧边栏宽度',
      mixCollapsedWidth: '混合布局侧边栏折叠宽度',
      mixChildMenuWidth: '混合布局子菜单宽度'
    },
    footer: {
      visible: '显示底部',
      fixed: '固定底部',
      height: '底部高度',
      right: '底部局右'
    },
    watermark: {
      visible: '显示全屏水印',
      text: '水印文本'
    },
    themeDrawerTitle: '主题配置',
    pageFunTitle: '页面功能',
    configOperation: {
      copyConfig: '复制配置',
      copySuccessMsg: '复制成功，请替换 src/theme/settings.ts 中的变量 themeSettings',
      resetConfig: '重置配置',
      resetSuccessMsg: '重置成功'
    }
  },
  route: {
    login: '登录',
    403: '无权限',
    404: '页面不存在',
    500: '服务器错误',
    'iframe-page': '外链页面',
    home: '首页',
    document: '文档',
    document_project: '项目文档',
    'document_project-link': '项目文档(外链)',
    document_vue: 'Vue文档',
    document_vite: 'Vite文档',
    document_unocss: 'UnoCSS文档',
    document_naive: 'Naive UI文档',
    document_antd: 'Ant Design Vue文档',
    'user-center': '个人中心',
    manage: '系统管理',
    manage_user: '用户管理',
    'manage_user-detail': '用户详情',
    manage_role: '角色管理',
    manage_menu: '菜单管理',
    manage_email: '邮箱配置',
    agent: 'Agent 管理',
    agent_config: '配置管理',
    agent_skills: 'Skills 管理',
    'agent_token-usage': 'Token 消耗',
    exception: '异常页',
    exception_403: '403',
    exception_404: '404',
    exception_500: '500'
  },
  button: {
    manage: {
      user: {
        list: '用户列表',
        add: '新增用户',
        delete: '删除用户',
        edit: '编辑用户'
      },
      menu: {
        list: '菜单列表',
        add: '新增菜单',
        delete: '删除菜单',
        edit: '编辑菜单'
      },
      role: {
        list: '角色列表',
        add: '新增角色',
        delete: '删除角色',
        edit: '编辑角色'
      }
    }
  },
  page: {
    login: {
      common: {
        loginOrRegister: '登录 / 注册',
        usernamePlaceholder: '请输入用户名',
        phonePlaceholder: '请输入手机号',
        codePlaceholder: '请输入验证码',
        emailPlaceholder: '请输入邮箱',
        passwordPlaceholder: '请输入密码',
        confirmPasswordPlaceholder: '请再次输入密码',
        codeLogin: '验证码登录',
        confirm: '确定',
        back: '返回',
        validateSuccess: '验证成功',
        registerSuccess: '注册成功',
        loginSuccess: '登录成功',
        welcomeBack: '欢迎回来，{username} ！'
      },
      pwdLogin: {
        title: '密码登录',
        rememberMe: '记住我',
        forgetPassword: '忘记密码？',
        register: '注册账号',
        otherAccountLogin: '其他账号登录',
        otherLoginMode: '其他登录方式',
        superAdmin: '超级管理员',
        admin: '管理员',
        user: '普通用户'
      },
      codeLogin: {
        emailType: '邮箱',
        phoneType: '手机号',
        title: '{type}验证码登录',
        getCode: '获取验证码',
        reGetCode: '{time}秒后重新获取',
        sendCodeSuccess: '验证码发送成功',
        imageCodePlaceholder: '请输入图片验证码'
      },
      register: {
        title: '注册账号',
        agreement: '我已经仔细阅读并接受',
        protocol: '《用户协议》',
        policy: '《隐私权政策》'
      },
      resetPwd: {
        title: '重置密码'
      },
      bindWeChat: {
        title: '绑定微信'
      }
    },
    home: {
      dashboard: {
        welcome: '欢迎回来 {username}',
        agentOnline: '运行中',
        agentOffline: '已停止',
        unavailable: '暂不可用',
        agentStatus: 'Agent 状态',
        currentModel: '当前模型',
        totalTokens: '累计 Token',
        installedSkills: '已安装 Skills'
      }
    },
    agentConfig: {
      title: '模型配置',
      lastRestart: '最近重启 {time}',
      restartAgent: '重启 agent',
      noRestartRecord: '暂无记录',
      supplier: '供应商',
      model: '模型',
      modelPlaceholder: '选择或输入模型 ID',
      reasoningEffort: '推理强度',
      defaultReasoningEffort: '使用模型默认值',
      saveAndRestart: '保存并重启 agent',
      status: {
        unknown: '未知',
        restarting: '重启中',
        running: '运行中',
        stopped: '未运行',
        inUse: '正在使用',
        configured: '已配置',
        pending: '待配置'
      },
      effort: {
        low: '轻度',
        medium: '中',
        high: '高',
        xhigh: '极高',
        max: 'Max',
        ultra: 'Ultra'
      },
      message: {
        selectProviderModel: '请选择供应商并填写模型',
        providerApiKeyRequired: '请输入 {provider} API Key',
        applied: '模型配置已应用，agent 已重启',
        restarted: 'agent 已重启',
        openAIApiKeyRequired: '请输入 OpenAI API Key',
        openAIApiKeySaved: 'OpenAI API Key 已保存',
        chatGPTLoginSuccess: 'ChatGPT 账号登录成功',
        openAILogoutSuccess: '已退出 OpenAI 账号'
      },
      openAI: {
        title: 'OpenAI 认证',
        chatGPTAccount: 'ChatGPT 账号',
        apiKey: 'OpenAI API Key',
        email: '邮箱',
        planType: '订阅类型',
        logout: '退出账号',
        logoutConfirm: '确认清除当前 ChatGPT 登录凭据？',
        apiKeyConfigured: 'API Key 已配置',
        clearApiKey: '清除 API Key',
        clearApiKeyConfirm: '确认清除当前 OpenAI API Key？',
        loginChatGPT: '登录 ChatGPT',
        useApiKey: '使用 API Key',
        saveAndUse: '保存并使用',
        backToChatGPT: '返回 ChatGPT 登录'
      },
      external: {
        title: '{provider} 配置',
        guide: '尚未配置 {provider}，请输入 API Key 后保存并重启。',
        apiAddress: 'API 地址',
        replaceKey: '输入新 Key 以替换当前配置',
        keyPlaceholder: '输入 {provider} API Key'
      },
      login: {
        title: '登录 ChatGPT',
        guide: '请按以下步骤完成登录，本页面会等待验证结果。',
        expired: '验证码已过期，请重新获取后继续。',
        openStep: '打开验证页面并登录 ChatGPT',
        openPage: '打开验证页面',
        codeStep: '在验证页面输入以下一次性验证码',
        copyCode: '复制',
        copySuccess: '验证码已复制',
        copyFailed: '复制失败，请手动复制',
        expiredLabel: '已过期',
        expiresIn: '有效期 {time}',
        waiting: '正在等待验证完成…',
        retry: '重新获取验证码'
      }
    },
    agentSkills: {
      title: 'Skills 管理',
      installed: '已安装 Skills',
      uploadArchive: 'Upload',
      searchPlaceholder: '搜索名称',
      deleteConfirm: '确认删除 {name}？',
      noMatch: '没有匹配的 Skill',
      empty: '暂无 Skills',
      editorPlaceholder: '编辑当前文件内容',
      loading: '加载中…',
      selectGuide: '选择左侧 Skill 查看内容'
    },
    agentTokenUsage: {
      summaryTitle: 'Token 概览',
      trendTitle: 'Token 消耗趋势',
      detailsTitle: '用量明细',
      input: '输入',
      cachedInput: '缓存输入',
      cacheWrite: '缓存写入',
      output: '输出',
      reasoningOutput: '推理输出',
      totalToken: '总 Token',
      tokenUnit: 'Token',
      byDay: '按天',
      byMonth: '按月',
      accountSearchPlaceholder: '输入用户名查询',
      conversationSearchPlaceholder: '输入 Conversation ID 查询',
      detailsTip: '账号分页展示，展开账号后按需加载对话和 Turn',
      emptyRecords: '暂无匹配的 Token 用量记录',
      unknownAccount: '未知账号',
      conversation: '对话',
      turn: '轮次',
      conversationCount: '共 {count} 个对话',
      emptyConversations: '暂无匹配的对话',
      lastUsed: '最近使用 {time}',
      seriesName: 'Token 消耗',
      turnId: 'Turn ID',
      time: '时间',
      contextWindow: '上下文窗口',
      accountCount: '共 {count} 个账号',
      quotaTitle: '额度设置',
      globalDefault: '全局默认',
      fiveHourQuota: '5 小时使用限额',
      sevenDayQuota: '每周使用限额',
      inputQuota: '请输入 Token 额度',
      quotaPositive: '两项额度都必须大于 0'
    },
    manage: {
      common: {
        status: {
          enable: '启用',
          disable: '禁用'
        }
      },
      role: {
        title: '角色列表',
        roleName: '角色名称',
        roleCode: '角色编码',
        roleStatus: '角色状态',
        roleDesc: '角色描述',
        menuAuth: '菜单权限',
        buttonAuth: '按钮权限',
        form: {
          roleName: '请输入角色名称',
          roleCode: '请输入角色编码',
          roleStatus: '请选择角色状态',
          roleDesc: '请输入角色描述'
        },
        addRole: '新增角色',
        editRole: '编辑角色'
      },
      user: {
        title: '用户列表',
        username: '用户名',
        nickName: '昵称',
        userPhone: '手机号',
        userEmail: '邮箱',
        userStatus: '用户状态',
        userRole: '用户角色',
        password: '密码',
        form: {
          username: '请输入用户名',
          nickName: '请输入昵称',
          userPhone: '请输入手机号',
          userEmail: '请输入邮箱',
          userStatus: '请选择用户状态',
          userRole: '请选择用户角色',
          password: '请输入密码'
        },
        addUser: '新增用户',
        editUser: '编辑用户'
      },
      menu: {
        home: '首页',
        title: '菜单列表',
        id: 'ID',
        parentId: '父级菜单ID',
        menuType: '菜单类型',
        menuName: '菜单名称',
        routeName: '路由名称',
        routePath: '路由路径',
        pathParam: '路径参数',
        layout: '布局',
        page: '页面组件',
        i18nKey: '国际化key',
        icon: '图标',
        localIcon: '本地图标',
        iconTypeTitle: '图标类型',
        order: '排序',
        constant: '常量路由',
        keepAlive: '缓存路由',
        href: '外链',
        iframePage: 'iframe page 链接',
        hideInMenu: '隐藏菜单',
        activeMenu: '高亮的菜单',
        multiTab: '支持多页签',
        fixedIndexInTab: '固定在页签中的序号',
        query: '路由参数',
        button: '按钮',
        buttonCode: '按钮编码',
        buttonDesc: '按钮描述',
        permission: '权限标识',
        permissionCode: '权限编码',
        permissionDesc: '权限描述',
        menuStatus: '菜单状态',
        form: {
          home: '请选择首页',
          menuType: '请选择菜单类型',
          menuName: '请输入菜单名称',
          routeName: '请输入路由名称',
          routePath: '请输入路由路径',
          pathParam: '请输入路径参数',
          page: '请选择页面组件',
          layout: '请选择布局组件',
          i18nKey: '请输入国际化key',
          icon: '请输入图标',
          localIcon: '请选择本地图标',
          order: '请输入排序',
          keepAlive: '请选择是否缓存路由',
          href: '请输入外链',
          hideInMenu: '请选择是否隐藏菜单',
          activeMenu: '请选择高亮的菜单的路由名称',
          multiTab: '请选择是否支持多标签',
          fixedInTab: '请选择是否固定在页签中',
          fixedIndexInTab: '请输入固定在页签中的序号',
          queryKey: '请输入路由参数Key',
          queryValue: '请输入路由参数Value',
          button: '请选择是否按钮',
          buttonCode: '请输入按钮编码',
          buttonDesc: '请输入按钮描述',
          permissionCode: '请输入权限编码',
          permissionDesc: '请输入权限描述',
          menuStatus: '请选择菜单状态'
        },
        addMenu: '新增菜单',
        editMenu: '编辑菜单',
        addChildMenu: '添加下一级',
        addButton: '新增按钮',
        type: {
          directory: '目录',
          menu: '菜单',
          button: '按钮'
        },
        iconType: {
          iconify: 'iconify图标',
          local: '本地图标'
        }
      }
    }
  },
  form: {
    required: '不能为空',
    username: {
      required: '请输入用户名',
      invalid: '用户名格式不正确'
    },
    phone: {
      required: '请输入手机号',
      invalid: '手机号格式不正确'
    },
    pwd: {
      required: '请输入密码',
      invalid: '密码格式不正确，6-18位字符，包含字母、数字、下划线'
    },
    confirmPwd: {
      required: '请输入确认密码',
      invalid: '两次输入密码不一致'
    },
    code: {
      required: '请输入验证码',
      invalid: '验证码格式不正确'
    },
    email: {
      required: '请输入邮箱',
      invalid: '邮箱格式不正确'
    }
  },
  dropdown: {
    closeCurrent: '关闭',
    closeOther: '关闭其它',
    closeLeft: '关闭左侧',
    closeRight: '关闭右侧',
    closeAll: '关闭所有'
  },
  icon: {
    themeConfig: '主题配置',
    themeSchema: '主题模式',
    lang: '切换语言',
    fullscreen: '全屏',
    fullscreenExit: '退出全屏',
    reload: '刷新页面',
    collapse: '折叠菜单',
    expand: '展开菜单',
    pin: '固定',
    unpin: '取消固定'
  },
  datatable: {
    itemCount: '共 {total} 条'
  }
};

export default local;
