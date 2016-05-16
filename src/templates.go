package main

// WebsiteTemplate Template to use for website index.html
const WebsiteTemplate = `<!doctype html>
<html lang="en" ng-app="myApp">
<head>
	<title><%Title%></title>
	<link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
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
				console.log("Test, " + fileName)
				var idx = fileName.lastIndexOf(".");
				return fileName.slice(0, idx) + "_thumb.jpg";
			}
		});
</script>
</head>
<body>
	<div class="container" ng-controller="MainCtrl">
		<a href="<%BACK%>"><%YEAR%></a><h2><%DATE%></h2>
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
<title><%Title%></title>
<link rel='stylesheet'  href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css' />
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
	<a href="<%Back%>">Back</a>
	<div class="container" ng-controller="MainCtrl">
		<h1><%Title%></h1>
		</br>
		<div class="navbar" />
		<div class="body">
			<div ng-repeat="date in dates">
				<div class="col-lg-3 col-md-4 col-xs-6 thumb">
					<p>{{date.date}}</p>
					<a href="{{date.date}}/index.html"><img ng-src="{{date.thumb}}" class="img-thumbnail" /></a>
				</div>
			</div>
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
