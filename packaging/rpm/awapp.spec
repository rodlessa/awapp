Name:           awapp
Version:        1.0.5
Release:        1
Summary:        Dependency-free terminal weather visualizer with ANSI animation
License:        MIT
URL:            https://github.com/rodlessa/awapp
Source0:        https://github.com/rodlessa/awapp/archive/refs/tags/v%{version}.tar.gz
BuildRequires:  golang >= 1.22
Requires:       glibc

%description
A terminal weather visualizer that renders current conditions as a
full-screen ANSI animation: sun/moon arcs, real star positions, seasonal
leaves, rain, clouds and more. Keyless Open-Meteo by default; optional
OpenWeatherMap key.

%prep
%setup -q -n awapp-%{version}

%build
go build -trimpath -ldflags="-s -w -X main.version=v%{version}" -o awapp .

%install
install -Dm755 awapp %{buildroot}%{_bindir}/awapp
install -Dm644 packaging/man/awapp.1 %{buildroot}%{_mandir}/man1/awapp.1

%check
./awapp -version | grep -q "v%{version}"

%files
%{_bindir}/awapp
%{_mandir}/man1/awapp.1*

%changelog
* Fri Aug 28 2026 rodlessa <rodlessa@users.noreply.github.com> - 1.0.5-1
- Initial RPM packaging
