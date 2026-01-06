#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试 Team 执行 - 验证修复后的 agno_runner.py
"""

import subprocess
import json
import os

def test_team_execution():
    """测试 team 执行"""
    
    # 构建测试配置（模拟后端传给 agno_runner.py 的数据）
    test_config = {
        "type": "team",
        "prompt": "请分析这段代码的问题：printf(123\");",
        "session_id": "test-session-1",
        "model": {
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "api_key": "user your own key",
            "base_url": "https://api.openai.com/v1"
        },
        "team": {
            "name": "test-team",
            "mode": "coordinate",
            "description": "Code review team",
            "leader": {
                "name": "manager",
                "personality": "你是一个技术经理，专注于代码规范的检查",
                "model": {
                    "provider": "openai",
                    "model_id": "gpt-4o-mini",
                    "api_key": "user your own key",
                    "base_url": "https://api.openai.com/v1"
                }
            },
            "members": [
                {
                    "name": "expert",
                    "personality": "你是一个研发专家，专注于关注性能优化",
                    "model": {
                        "provider": "openai",
                        "model_id": "gpt-4o-mini",
                        "api_key": "user your own key",
                        "base_url": "https://api.openai.com/v1"
                    }
                }
            ]
        },
        "context": {},
        "stream": True,
        "debug": False,
        "reuse_session": False
    }
    
    print("=" * 100)
    print("测试 Team 执行（带代理配置）")
    print("=" * 100)
    print()
    
    # 设置代理环境变量
    env = os.environ.copy()
    env["HTTP_PROXY"] = "http://127.0.0.1:7890"
    env["HTTPS_PROXY"] = "http://127.0.0.1:7890"
    env["http_proxy"] = "http://127.0.0.1:7890"
    env["https_proxy"] = "http://127.0.0.1:7890"
    
    # 调用 agno_runner.py
    script_path = "scripts/agno_runner.py"
    
    try:
        proc = subprocess.Popen(
            ["python3", script_path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=env
        )
        
        # 发送配置
        stdout, stderr = proc.communicate(input=json.dumps(test_config), timeout=30)
        
        print("【标准输出】")
        print("-" * 100)
        
        # 解析事件
        has_content = False
        has_error = False
        events = []
        
        for line in stdout.strip().split("\n"):
            if not line:
                continue
            try:
                event = json.loads(line)
                events.append(event)
                
                event_type = event.get("type")
                content = event.get("content", "")
                
                if event_type == "error":
                    has_error = True
                    print(f"❌ 错误: {content}")
                elif event_type == "warning":
                    print(f"⚠️  警告: {content}")
                elif event_type == "content":
                    has_content = True
                    print(f"✅ 内容: {content[:100]}...")
                elif event_type == "completed":
                    print(f"✅ 完成: {content[:100] if content else '(empty)'}")
                else:
                    print(f"📝 {event_type}: {content[:80] if content else ''}")
                    
            except json.JSONDecodeError:
                print(f"⚠️  非JSON输出: {line[:100]}")
        
        print()
        print("-" * 100)
        
        if stderr:
            print("\n【标准错误】")
            print("-" * 100)
            print(stderr)
            print("-" * 100)
        
        # 分析结果
        print("\n" + "=" * 100)
        print("测试结果分析:")
        print("=" * 100)
        print(f"总事件数: {len(events)}")
        print(f"是否有内容输出: {'✅ 是' if has_content else '❌ 否'}")
        print(f"是否有错误: {'❌ 是' if has_error else '✅ 否'}")
        print(f"进程退出码: {proc.returncode}")
        
        if not has_content and not has_error:
            print("\n⚠️  没有内容输出且没有错误，可能是静默失败")
            print("建议检查：")
            print("  1. API Key 是否有效")
            print("  2. 代理是否正常工作")
            print("  3. 网络连接是否正常")
        
        return proc.returncode == 0 and has_content
        
    except subprocess.TimeoutExpired:
        print("❌ 测试超时")
        proc.kill()
        return False
    except Exception as e:
        print(f"❌ 测试异常: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_team_execution()
    exit(0 if success else 1)
