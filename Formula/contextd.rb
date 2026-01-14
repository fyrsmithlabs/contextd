# typed: false
# frozen_string_literal: true

class Contextd < Formula
  desc "AI context and reasoning engine for Claude Code"
  homepage "https://github.com/fyrsmithlabs/contextd"
  version "0.3.4"
  license "AGPL-3.0"

  url "https://github.com/fyrsmithlabs/contextd/archive/refs/tags/v0.3.4.tar.gz"
  sha256 "4aee0cb6608bc12ad14e9baced30907ce7225574b79191800fd93e56d1bb54c1"

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
