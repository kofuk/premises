import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

const i18nData = {
    en: {
        translation: {
            app_name: 'Control Panel',
            logout: 'Logout',
            title_login: 'Login',
            username: 'User',
            password: 'Password',
            login: 'Login',
            logging_in: 'Logging in…',
            next: 'Next',
            back: 'Back',
            refresh: 'Refresh',
            copy: 'Copy',
            launch_server: 'Start',
            relaunch_server: 'Restart',
            stop_server: 'Stop',
            config_choose_backup: 'World',
            config_configure_world: 'Configure World',
            config_server_version: 'Game Version',
            config_machine_type: 'Machine Type',
            config_world_name: 'World Name',
            config_world_source: 'World Source',
            world_type_default: 'Default',
            world_type_superflat: 'Superflat',
            world_type_large_biomes: 'Large Biomes',
            world_type_amplified: 'Amplified',
            no_backups: 'No world backups found. Please generate a new world.',
            use_cached_world: 'Use Cached World Data If Possible',
            backup_generation: 'Backup Generation',
            select_world: 'World',
            world_type: 'World Type',
            seed: 'Seed',
            version_show_stable: 'Show Stable',
            version_show_snapshot: 'Show Snapshot',
            version_show_beta: 'Show Beta',
            version_show_alpha: 'Show Alpha',
            world_name: 'World Name',
            use_backups: 'Use backups',
            generate_world: 'Generate a New World',
            menu_world_info: 'World Information',
            menu_reconfigure: 'Reconfigure Server',
            menu_snapshot: 'Snapshot',
            menu_system_info: 'System Information',
            system_info_server_version: 'Server Version',
            system_info_host_os: 'Host OS',
            snapshot_description: 'This saves snapshot of playing world. After snapshot saved, you can reconfigure the server using saved data.',
            requesting: 'Requesting…',
            take_snapshot: 'Take Snapshot',
            world_info_game_version: 'Game Version',
            world_info_world_name: 'World',
            world_info_seed: 'Seed',
            connected: 'Connected',
            disconnected: 'Connection has lost\nPlease reload the page',
            reconnecting: 'Connection has lost\nReconnecting…',
            connecting: 'Connecting…',
            notification_toast_title: 'Want to receive notification?',
            notification_toast_description: 'You can receive notification when the server is ready.',
            notification_allow: 'Allow notification',
            notification_title: 'Minecraft server is launched!',
            notification_body: 'You can login to the game.',
            title_setup: 'Set up',
            setup_new_user_description: "Let's create first user!",
            setup_continue: 'Continue',
            password_confirm: 'Confirm password',
            settings: 'Settings',
            change_password_header: 'Change Password',
            change_password_current: 'Current Password',
            change_password_new: 'New Password',
            change_password_confirm: 'Confirm New Password',
            change_password_success: 'Password changed',
            change_password_submit: 'Change',
            add_user_header: 'Add User',
            add_user_success: 'User added',
            add_user_submit: 'Add',
            set_password_title: 'Set Password',
            set_password_submit: 'Set password'
        }
    },
    ja: {
        translation: {
            app_name: 'コントロールパネル',
            logout: 'ログアウト',
            title_login: 'ログイン',
            username: 'ユーザー名',
            password: 'パスワード',
            login: 'ログイン',
            logging_in: 'ログイン中…',
            next: '次へ',
            back: '戻る',
            refresh: '再読み込み',
            copy: 'コピー',
            launch_server: '起動',
            relaunch_server: '再起動',
            stop_server: '停止',
            config_choose_backup: 'ワールド',
            config_configure_world: 'ワールドの構成',
            config_server_version: 'ゲームのバージョン',
            config_machine_type: 'マシンタイプ',
            config_world_name: 'ワールド名',
            config_world_source: 'ワールドの読み込み先',
            world_type_default: 'デフォルト',
            world_type_superflat: 'スーパーフラット',
            world_type_large_biomes: '大きなバイオーム',
            world_type_amplified: 'アンプリファイド',
            no_backups: 'バックアップが見つかりません。新しいワールドを生成してください。',
            use_cached_world: '可能ならキャッシュされたワールドのデータを使用',
            backup_generation: 'バックアップの世代',
            select_world: 'ワールド',
            world_type: 'ワールドのタイプ',
            seed: 'シード値',
            version_show_stable: '安定版を表示する',
            version_show_snapshot: 'スナップショットを表示する',
            version_show_beta: 'ベータ版を表示する',
            version_show_alpha: 'アルファ版を表示する',
            world_name: 'ワールド名',
            use_backups: 'バックアップを使用',
            generate_world: '新しい世界を生成',
            menu_world_info: 'ワールド情報',
            menu_reconfigure: 'サーバを再設定',
            menu_snapshot: 'ワールドのスナップショット',
            menu_system_info: 'システム情報',
            system_info_server_version: 'サーバのバージョン',
            system_info_host_os: 'ホストのOS',
            snapshot_description:
                'これによりワールドのスナップショットが保存されます。スナップショット保存の終了後、”サーバの再設定”メニューからスナップショットのデータを使用してサーバを再構築できます。',
            requesting: 'リクエストを送信しています…',
            take_snapshot: 'スナップショットを作成',
            world_info_game_version: 'ゲームのバージョン',
            world_info_world_name: 'ワールド',
            world_info_seed: 'シード値',
            connected: '接続済み',
            disconnected: '切断されました\nページをリロードして再接続してください',
            reconnecting: '切断されました\n再接続しています…',
            connecting: '接続しています…',
            notification_toast_title: '通知を受け取りますか？',
            notification_toast_description: '通知を許可するとサーバが起動したときに通知を受け取れます。',
            notification_allow: '通知を許可',
            notification_title: 'Minecraft サーバが起動しました！',
            notification_body: 'ゲームにログインできます',
            title_setup: 'セットアップ',
            setup_new_user_description: '最初のユーザを作成しましょう！',
            setup_continue: '続行',
            password_confirm: 'パスワードを確認',
            settings: '設定',
            change_password_header: 'パスワードを変更',
            change_password_current: '現在のパスワード',
            change_password_new: '新しいパスワード',
            change_password_confirm: '新しいパスワードを確認',
            change_password_success: 'パスワードを変更しました',
            change_password_submit: '変更',
            add_user_header: 'ユーザを追加',
            add_user_success: 'ユーザを追加しました',
            add_user_submit: '追加',
            set_password_title: 'パスワードを設定',
            set_password_submit: 'パスワードを設定'
        }
    }
};

export default i18n.use(LanguageDetector).init({
    resources: i18nData,
    debug: true,
    interpolation: {
        escapeValue: false
    }
});
