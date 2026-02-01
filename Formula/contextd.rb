# typed: false
# frozen_string_literal: true

class Contextd < Formula
  desc "AI context and reasoning engine for Claude Code"
  homepage "https://github.com/fyrsmithlabs/contextd"
  version "0.4.0"
  license "AGPL-3.0"

  url "https://github.com/fyrsmithlabs/contextd/archive/refs/tags/v0.4.0.tar.gz"
  sha256 "9f717428e0d3baba7df76e29d5cd1152e4ed7709618c20b4b6746e2aa6ebff58"

  depends_on "go" => :build
  depends_on "onnxruntime"

  def install
    ENV["CGO_ENABLED"] = "1"

    ldflags = %W[
      -s -w
      -X main.version=#{version}
    ]

    system "go", "build", *std_go_args(ldflags:), "-o", bin/"contextd", "./cmd/contextd"
    system "go", "build", *std_go_args(ldflags:), "-o", bin/"ctxd", "./cmd/ctxd"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/contextd --version")
    assert_match version.to_s, shell_output("#{bin}/ctxd --version")
  end
end
