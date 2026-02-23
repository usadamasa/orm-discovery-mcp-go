#!/usr/bin/env python3
"""
O'Reilly Login Diagnosis Script

診断専用スクリプト - プロダクションコードではありません
Playwright を使用してログインフローの各ステップを可視化し、
失敗箇所を特定します。

Usage:
    OREILLY_USER_ID="email@acm.org" OREILLY_PASSWORD="password" python scripts/diagnose_login.py
    OREILLY_USER_ID="email@acm.org" OREILLY_PASSWORD="password" python scripts/diagnose_login.py --headless

Prerequisites:
    pip install playwright
    python -m playwright install chromium
"""

import os
import sys
import argparse
from pathlib import Path
from datetime import datetime


def main():
    parser = argparse.ArgumentParser(description="O'Reilly login flow diagnosis")
    parser.add_argument("--headless", action="store_true", help="Run in headless mode")
    parser.add_argument(
        "--output-dir",
        default="/tmp/orm-diagnosis",
        help="Directory for screenshots (default: /tmp/orm-diagnosis)",
    )
    args = parser.parse_args()

    user_id = os.environ.get("OREILLY_USER_ID", "")
    password = os.environ.get("OREILLY_PASSWORD", "")

    if not user_id or not password:
        print("[ERROR] OREILLY_USER_ID and OREILLY_PASSWORD environment variables are required")
        sys.exit(1)

    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

    print(f"[INFO] Output directory: {output_dir}")
    print(f"[INFO] Headless mode: {args.headless}")
    print(f"[INFO] User: {user_id}")
    print()

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("[ERROR] Playwright not installed. Run: pip install playwright && python -m playwright install chromium")
        sys.exit(1)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=args.headless)
        context = browser.new_context(
            viewport={"width": 1280, "height": 720},
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        )
        page = context.new_page()

        def screenshot(step_name, description=""):
            path = output_dir / f"{timestamp}_{step_name}.png"
            page.screenshot(path=str(path))
            print(f"  [SCREENSHOT] {path}")
            if description:
                print(f"  [DESC] {description}")

        def log_step(step, message):
            print(f"\n=== Step {step}: {message} ===")
            print(f"  [URL] {page.url}")

        # Step 1: ログインページに移動
        log_step(1, "ログインページへ移動")
        try:
            page.goto("https://www.oreilly.com/member/login/", wait_until="domcontentloaded", timeout=30000)
            screenshot("01_login_page", "ログインページ初期状態")
            print(f"  [OK] ページ読み込み成功")
            print(f"  [TITLE] {page.title()}")
        except Exception as e:
            screenshot("01_login_page_error", f"エラー: {e}")
            print(f"  [ERROR] ページ読み込み失敗: {e}")
            _print_body_excerpt(page)
            browser.close()
            return

        # Access Denied チェック
        body_text = page.evaluate("() => document.body.innerText || ''")
        if "access denied" in body_text.lower() or "reference #" in body_text.lower():
            print("  [AKAMAI] Access Denied 検出 → Akamai Bot Manager がブロックしています")
            print("  [ACTION] 対処: Cookie-first 運用に切り替えるか、時間を置いて再試行してください")
            browser.close()
            return

        # Step 2: email フィールドを探す
        log_step(2, "email フィールドの検出")
        email_selectors = [
            'input[name="email"]',
            'input[type="email"]',
            'input[placeholder*="email" i]',
            'input[id*="email" i]',
            '#email',
        ]
        found_email_selector = None
        for sel in email_selectors:
            try:
                el = page.wait_for_selector(sel, timeout=3000)
                if el:
                    found_email_selector = sel
                    print(f"  [OK] email フィールド発見: {sel}")
                    break
            except Exception:
                print(f"  [NOT FOUND] {sel}")

        if not found_email_selector:
            print("  [ERROR] email フィールドが見つかりません → セレクタ変更の可能性")
            _print_form_inputs(page)
            screenshot("02_email_not_found", "email フィールド未発見")
            browser.close()
            return

        # email を入力
        page.fill(found_email_selector, user_id)
        screenshot("02_email_filled", f"email 入力完了 (selector: {found_email_selector})")
        print(f"  [OK] email 入力: {user_id}")

        # Step 3: Continue ボタンを探してクリック
        log_step(3, "Continue ボタンのクリック")
        button_selectors = [
            '.orm-Button-root',
            'button[type="submit"]',
            'button[data-testid*="continue" i]',
            'button:has-text("Continue")',
            'button:has-text("Sign in")',
            'input[type="submit"]',
        ]
        found_button_selector = None
        for sel in button_selectors:
            try:
                el = page.wait_for_selector(sel, timeout=3000)
                if el and el.is_visible():
                    found_button_selector = sel
                    print(f"  [OK] ボタン発見: {sel}")
                    break
            except Exception:
                print(f"  [NOT FOUND] {sel}")

        if not found_button_selector:
            print("  [ERROR] Continue ボタンが見つかりません → セレクタ変更の可能性")
            _print_buttons(page)
            screenshot("03_button_not_found", "Continue ボタン未発見")
            browser.close()
            return

        page.click(found_button_selector)
        screenshot("03_continue_clicked", f"Continue クリック後 (selector: {found_button_selector})")
        print(f"  [OK] ボタンクリック: {found_button_selector}")

        # Step 4: リダイレクト後 URL の確認
        log_step(4, "リダイレクト後の URL 確認")
        try:
            # ページ遷移またはDOM変化を待機
            page.wait_for_load_state("domcontentloaded", timeout=15000)
        except Exception as e:
            print(f"  [WARN] 遷移待機タイムアウト: {e}")

        current_url = page.url
        screenshot("04_after_redirect", f"リダイレクト後 URL: {current_url}")
        print(f"  [URL] リダイレクト後: {current_url}")

        if "idp.acm.org" in current_url:
            print("  [OK] ACM IDP にリダイレクトされました")
            is_acm_idp = True
        elif "learning.oreilly.com" in current_url or "oreilly.com/home" in current_url:
            print("  [OK] O'Reilly ホームに直接遷移 → すでにログイン済みの可能性")
            browser.close()
            return
        else:
            print(f"  [WARN] 想定外のリダイレクト先: {current_url}")
            print("  [INFO] 期待値: idp.acm.org へのリダイレクト")
            is_acm_idp = False

        # Step 5: ACM IDP での認証
        if is_acm_idp:
            log_step(5, "ACM IDP でのログイン")
            screenshot("05_acm_idp_initial", "ACM IDP 初期状態")

            username = user_id.removesuffix("@acm.org")
            print(f"  [INFO] ACM ユーザー名: {username}")

            # username フィールドを探す
            acm_username_selectors = [
                'input[placeholder*="username" i]',
                'input[name="username"]',
                'input[id*="username" i]',
                'input[type="text"]',
            ]
            found_acm_username = None
            for sel in acm_username_selectors:
                try:
                    el = page.wait_for_selector(sel, timeout=5000)
                    if el:
                        found_acm_username = sel
                        print(f"  [OK] ACM username フィールド発見: {sel}")
                        break
                except Exception:
                    print(f"  [NOT FOUND] {sel}")

            if not found_acm_username:
                print("  [ERROR] ACM username フィールドが見つかりません → セレクタ変更の可能性")
                _print_form_inputs(page)
                screenshot("05_acm_username_not_found", "ACM username フィールド未発見")
                browser.close()
                return

            # username と password を入力
            page.fill(found_acm_username, username)

            acm_password_selectors = [
                'input[placeholder*="password" i]',
                'input[name="password"]',
                'input[type="password"]',
            ]
            found_acm_password = None
            for sel in acm_password_selectors:
                try:
                    el = page.wait_for_selector(sel, timeout=3000)
                    if el:
                        found_acm_password = sel
                        print(f"  [OK] ACM password フィールド発見: {sel}")
                        break
                except Exception:
                    print(f"  [NOT FOUND] {sel}")

            if not found_acm_password:
                print("  [ERROR] ACM password フィールドが見つかりません")
                screenshot("05_acm_password_not_found", "ACM password フィールド未発見")
                browser.close()
                return

            page.fill(found_acm_password, password)
            screenshot("05_acm_filled", "ACM 認証情報入力完了")
            print("  [OK] ACM 認証情報入力完了")

            # Sign In ボタンをクリック
            acm_button_selectors = [
                '.btn',
                'button[type="submit"]',
                'input[type="submit"]',
                'button:has-text("Sign in")',
                'button:has-text("Login")',
            ]
            found_acm_button = None
            for sel in acm_button_selectors:
                try:
                    el = page.wait_for_selector(sel, timeout=3000)
                    if el and el.is_visible():
                        found_acm_button = sel
                        print(f"  [OK] ACM Sign In ボタン発見: {sel}")
                        break
                except Exception:
                    print(f"  [NOT FOUND] {sel}")

            if found_acm_button:
                page.click(found_acm_button)
                print(f"  [OK] ACM Sign In クリック: {found_acm_button}")

                # ログイン完了待機
                try:
                    page.wait_for_url("**/learning.oreilly.com/**", timeout=30000)
                    screenshot("06_login_success", "ログイン成功")
                    print(f"\n  [SUCCESS] ログイン成功! URL: {page.url}")
                except Exception:
                    final_url = page.url
                    screenshot("06_login_result", f"最終 URL: {final_url}")
                    print(f"  [RESULT] 最終 URL: {final_url}")
                    if "learning.oreilly.com" in final_url or "oreilly.com/home" in final_url:
                        print("  [SUCCESS] ログイン成功!")
                    else:
                        print("  [ERROR] ログイン未完了")
                        _print_body_excerpt(page)
            else:
                print("  [ERROR] ACM Sign In ボタンが見つかりません")
                _print_buttons(page)
                screenshot("05_acm_button_not_found", "ACM Sign In ボタン未発見")

        else:
            # ACM IDP 以外のリダイレクト先
            print("\n  [INFO] ACM IDP 以外のリダイレクト先 → 新しい認証フローの可能性")
            print("  [INFO] ページのフォーム要素を調査します...")
            _print_form_inputs(page)
            _print_buttons(page)

        print(f"\n=== 診断完了 ===")
        print(f"スクリーンショット: {output_dir}/{timestamp}_*.png")
        browser.close()


def _print_body_excerpt(page):
    """ページ本文の先頭を表示"""
    try:
        text = page.evaluate("() => document.body.innerText?.substring(0, 500) || ''")
        print(f"  [BODY EXCERPT]\n{text}")
    except Exception:
        pass


def _print_form_inputs(page):
    """フォーム内の input 要素を列挙"""
    try:
        inputs = page.evaluate("""() => {
            return Array.from(document.querySelectorAll('input')).map(el => ({
                type: el.type,
                name: el.name,
                id: el.id,
                placeholder: el.placeholder,
                className: el.className.substring(0, 80)
            }));
        }""")
        print(f"  [FORM INPUTS] 発見した input 要素:")
        for inp in inputs:
            print(f"    type={inp['type']} name={inp['name']} id={inp['id']} placeholder={inp['placeholder']}")
    except Exception as e:
        print(f"  [WARN] input 要素の取得に失敗: {e}")


def _print_buttons(page):
    """ページ内のボタン要素を列挙"""
    try:
        buttons = page.evaluate("""() => {
            return Array.from(document.querySelectorAll('button, input[type="submit"]')).map(el => ({
                tag: el.tagName,
                type: el.type || '',
                text: (el.innerText || el.value || '').substring(0, 50),
                className: el.className.substring(0, 80)
            }));
        }""")
        print(f"  [BUTTONS] 発見したボタン要素:")
        for btn in buttons:
            print(f"    <{btn['tag']}> type={btn['type']} text='{btn['text']}' class={btn['className']}")
    except Exception as e:
        print(f"  [WARN] ボタン要素の取得に失敗: {e}")


if __name__ == "__main__":
    main()
