class Awapp < Formula
  desc "Dependency-free terminal weather visualizer with ANSI animation"
  homepage "https://github.com/rodlessa/awapp"
  url "https://github.com/rodlessa/awapp/archive/refs/tags/v1.0.5.tar.gz"
  sha256 "REPLACE_WITH_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-ldflags", "-s -w -X main.version=#{version}", "-o", bin/"awapp", "."
    man1.install "packaging/man/awapp.1"
    bash_completion.install "packaging/completions/awapp.bash" => "awapp"
    fish_completion.install "packaging/completions/awapp.fish"
    zsh_completion.install "packaging/completions/awapp.zsh" => "_awapp"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/awapp -version")
  end
end
