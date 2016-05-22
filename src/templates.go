package main

// WebsiteTemplate Template to use for website index.html
const WebsiteTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
	<title><%Title%></title>
	<link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
	<style>
		body { background-color: ghostwhite; }
		h2 { padding-left: 30px; }
		ul { list-style-type: none; }
		.header { padding: 20px}
		.img-thumbnail { height: 140px; }
		.caption { padding-left: 45px; }
	</style>
	<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
	<script type="text/javascript">
		var myApp = angular.module('myApp',[]);

		myApp.controller("MainCtrl", function($scope, $http, $q) {
			var res = $http.get("photos.json").then(function successCallback(results) {
				$scope.files = results.data.files;
			}, function errorCallback(response) {
				alert(response)
			})

			// gets thethumbnail name for the file
			$scope.getThumbJpg = function(fileName) {
				var idx = fileName.lastIndexOf(".");
				return fileName.slice(0, idx) + "_thumb.jpg";
			}
		});
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<div class="header">
			<a class="h2"href="<%BACK%>"><%YEAR%>/</a>
			<span class="h2"><%DATE%></h2>
		</div>
		<div class="body">
			<div ng-repeat="filename in files">
				<div class="col-lg-3 col-md-4 col-xs-6 thumb">
					<a href="{{filename}}"><img ng-src="{{getThumbJpg(filename)}}" class="img-thumbnail" alt="{{filename}}"/></a>
				</div>
			</div>
		</div>
	</div>
</body>
</html>`

// FolderTemplate Template to use for folder index.html
const FolderTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
<title><%TITLE%></title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css" integrity="sha384-1q8mTJOASx8j1Au+a5WDVnPi2lkFfwwEAa8hDDdjZlpLegxhjVME1fgjWPGmkzs7" crossorigin="anonymous">
<style>
body { background-color: ghostwhite; }
h2 { padding-left: 30px; }
ul { list-style-type: none; }
.img-thumbnail { height: 140px; }
.caption { padding-left: 45px; }
</style>

<script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
<script type="text/javascript">
	var myApp = angular.module('myApp',[]);

	myApp.controller("MainCtrl", function($scope, $http, $q) {
		var res = $http.get("dates.json").then(function successCallback(results) {
			$scope.dates = results.data.dates;
		}, function errorCallback(response) {
			alert(response)
		})
	});
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<h2><%TITLE%></h1>
		<a class="h2" href="../index.html">BACK/</a>
		</br>
		<div class="body">
			<ul id="image-list" class="row">
				<li ng-repeat="date in dates" class="col-lg-2 col-md-4 col-sm-6">
					<a href="{{date.date}}/index.html"><img ng-src="{{date.thumb}}" class="img-thumbnail" /></a>
					<span class="caption">{{date.date}}</span>
				</li>
			</ul>
		</div>
	</div>
</div>
</body>
</html>`

// MainTemplate Template to use for main root index.html
const MainTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
  <title><%Title%></title>
  <link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
  <script src="https://ajax.googleapis.com/ajax/libs/angularjs/1.4.8/angular.js"></script>
  <script type="text/javascript">
    var myApp = angular.module('myApp',[]);

    myApp.controller("MainCtrl", function($scope, $http, $q) {
      var res = $http.get("years.json").then(function successCallback(results) {
        $scope.years = results.data.years;
      }, function errorCallback(response) {
        alert(response)
      })
    });
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<h1><%Title%></h1>
		</br>
		<div class="navbar" />
		<div class="body">
			<div ng-repeat="year in years">
				<div class="col-lg-3 col-md-4 col-xs-6 thumb">
					<p>{{year}}</p>
					<a href="{{year}}/index.html"><img ng-src="http://findicons.com/files/icons/2221/folder/128/normal_folder.png" class="img-thumbnail" /></a>
				</div>
			</div>
		</div>
  </div>
</body>
</html>`
